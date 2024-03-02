package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v2"
)

const (
	dbBucket     = "app"
	syncInterval = 30
)

type Repository struct {
	URL            string
	Ref            string
	Username       string
	Password       string
	PrivateKeyFile string
}

const (
	GoSDK     = "go"
	NodeSDK   = "node"
	PythonSDK = "python"
)

type Config struct {
	Module     string   `yaml:"module"`
	Function   string   `yaml:"function"`
	Flags      []string `yaml:"flags"`
	WithSource bool     `yaml:"withSource"`
}

func (config *Config) Load(src string) error {
	data, err := os.ReadFile(filepath.Join(src, "cacidy.yaml"))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &config)
}

func NewConfig(src string) (*Config, error) {
	conf := &Config{}
	if err := conf.Load(src); err != nil {
		return nil, err
	}
	return conf, nil
}

type Runner struct {
	Repository *Repository `json:"repository"`
}

func (r *Runner) gitAuth() (transport.AuthMethod, error) {
	var auth transport.AuthMethod
	if r.Repository.PrivateKeyFile != "" {
		var err error
		if _, err := os.Stat(r.Repository.PrivateKeyFile); err != nil {
			return nil, errors.New("private key not found")
		}
		auth, err = ssh.NewPublicKeysFromFile("git", r.Repository.PrivateKeyFile, "")
		if err != nil {
			return nil, errors.New("failed while parsing the private key")
		}
	}
	if r.Repository.Password != "" {
		auth = &http.BasicAuth{
			Username: r.Repository.Username,
			Password: r.Repository.Password,
		}
	}
	return auth, nil
}

func (r *Runner) Clone(dir string) error {
	if _, err := os.Stat(r.Repository.PrivateKeyFile); err != nil {
		return err
	}
	auth, err := r.gitAuth()
	if err != nil {
		return err
	}
	if _, err := git.PlainClone(dir, false, &git.CloneOptions{
		Auth:     auth,
		URL:      r.Repository.URL,
		Progress: os.Stdout,
	}); err != nil {
		return fmt.Errorf("failed cloning the application: %s", err)
	}
	return nil
}

func (r *Runner) Checksum() (string, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{r.Repository.URL},
	})
	auth, err := r.gitAuth()
	if err != nil {
		return "", err
	}
	refs, err := remote.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return "", fmt.Errorf("failed getting the application checksum: %s", err)
	}
	for _, ref := range refs {
		if ref.Name().String() == fmt.Sprintf("refs/heads/%s", r.Repository.Ref) {
			return ref.Hash().String()[:7], nil
		}
	}
	return "", errors.New("checksum not found")
}

func (r *Runner) LoadChecksum(dbFile string) (string, error) {
	var checksum string
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return "", nil
	}
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return "", err
	}
	defer db.Close()
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dbBucket))
		if b == nil {
			return nil
		}
		data := b.Get([]byte("checksum"))
		if data == nil {
			return errors.New("state is empty")
		}
		checksum = string(data)
		return nil
	}); err != nil {
		return "", nil
	}
	return checksum, nil
}

func (r *Runner) SaveChecksum(dbFile, checksum string) error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(dbBucket))
		if err != nil {
			return err
		}
		return b.Put([]byte("checksum"), []byte(checksum))
	})
}

type callCommand struct {
	source   string
	module   string
	function string
	flags    []string
}

func (c *callCommand) render() string {
	cmd := []string{"call"}
	if c.module != "" {
		cmd = append(cmd, "-m", c.module)
	}
	if os.Getenv("CACIDY_DEBUG") != "" {
		cmd = append(cmd, "--debug")
	}
	progress := os.Getenv("CACIDY_PROGRESS")
	if progress != "" {
		cmd = append(cmd, "--progress", string(progress))
	}
	cmd = append(cmd, "--source", c.source)
	if len(c.flags) > 0 {
		cmd = append(cmd, c.flags...)
	}
	if c.function != "" {
		cmd = append(cmd, c.function)
	}
	return strings.Join(cmd, " ")
}

func (r *Runner) runPipeline(source, module, function string, flags []string) error {
	c := &callCommand{source, module, function, flags}
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("dagger %s", c.render()))
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func RunPipeline(source, module, function string) error {
	config, err := NewConfig(source)
	if err != nil {
		return err
	}
	if module == "" {
		module = config.Module
	}
	if function == "" {
		function = config.Function
	}
	return (&Runner{}).runPipeline(source, module, function, config.Flags)
}

// Start
func Listen(dataDir string, repo *Repository) error {
	if repo.URL == "" || repo.Ref == "" {
		return errors.New("git repository and branch are required")
	}
	dbFile := filepath.Join(dataDir, "bolt.db")
	r := &Runner{repo}
	log.Info("Starting listener")
	for {
		checksum, err := r.Checksum()
		if err != nil {
			return err
		}
		currentChecksum, err := r.LoadChecksum(dbFile)
		if err != nil {
			return err
		}
		if currentChecksum != "" && checksum != currentChecksum {
			log.Info("[%s] starting pipeline...\n", checksum)
			src, err := os.MkdirTemp("", "src")
			if err != nil {
				return err
			}
			if err := r.Clone(src); err != nil {
				return err
			}
			config, err := NewConfig(src)
			if err != nil {
				return err
			}
			err = r.runPipeline(src, config.Module, config.Function, config.Flags)
			if err := os.RemoveAll(src); err != nil {
				return err
			}
			if err != nil {
				return err
			}
		}
		if err := r.SaveChecksum(dbFile, checksum); err != nil {
			return err
		}
		time.Sleep(time.Second * syncInterval)
	}
}

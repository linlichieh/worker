package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"
)

// FIXME
var Queue chan string

type managerConfig struct {
	Env        EnvConfig
	Containers []ContainerConfig
}

type EnvConfig struct {
	Env string
}

type ContainerConfig struct {
	Name        string
	Provider    string
	Endpoint    string
	Source      string
	Concurrency int
	Enabled     bool
}

// ProjectName
type manager struct {
	workers map[string]*worker
	config  *managerConfig

	// TODO
	sqs struct {
		VisibleChan chan *Job
	}
	doneChan chan *Job

	// TODO
	log *io.Writer
}

func init() {
	// FIXME
	Queue = make(chan string)
}

func New() (*manager, error) {
	var m manager
	m.workers = make(map[string]*worker)
	m.doneChan = make(chan *Job)
	return &m, nil
}

// Initialisation with config file
func (m *manager) SetConfigWithFile(path string) (err error) {
	// TODO Read config file into config struct
	// Config validation
	// if len(configs) == 0 {
	//	return nil, errors.New("Can not load config")
	// }
	// for _, c := range configs {
	//	if err := c.validate(); err != nil {
	//		return nil, err
	//	}
	// }
	m.config = &managerConfig{}
	return
}

// Initialisation with config in json
func (m *manager) SetConfigWithJSON(json string) (err error) {
	// TODO Read json string into config struct
	m.config = &managerConfig{}
	return
}

func (m *manager) Run() {
	if m.config == nil {
		log.Fatal("Please set config before running")
	}
	for _, c := range m.config.Containers {
		// New worker
		w := newWorker(&workerConfig{
			Env:       m.config.Env,
			Container: c,
		})
		w.doneChan = m.doneChan
		w.run()
		m.workers[c.Name] = &w

		// Receive messages
		go m.receive(&c)
	}
	go m.done()
}

// TODO SQS Receive should be in package
func (m *manager) receive(c *ContainerConfig) { // TODO pass config
	var err error
	for {
		body := <-Queue // TODO Received
		var j Job
		if err = json.Unmarshal([]byte(body), &j.Desc); err != nil {
			log.Printf("Wrong job format: %s\n", body)
			// TODO Remove msg from queue
			continue
		}
		if err = j.validate(); err != nil {
			log.Printf("Wrong job format: %s\n", body)
			// TODO Remove msg from queue
			continue
		}
		// FIXME
		fmt.Println("Receive: " + body)

		j.receivedAt = time.Now()
		m.workers[c.Name].receivedChan <- &j
	}
}

func (m *manager) done() {
	for {
		j := <-m.doneChan
		fmt.Printf("Job done, Duration: %.1fs, ContainerName: %s, Type: %s, ID: %s\n", j.duration.Seconds(), j.Config.Container.Name, j.Desc.JobType, j.Desc.JobID)

		// TODO Graceful shutdown
	}
}

// New job type
func (m *manager) NewJobType(j JobBehaviour, containerName string, jobType string) {
	if containerName == "" || jobType == "" {
		log.Fatal("Both container name and job type cannot be empty")
	}
	if reflect.ValueOf(j).Kind() == reflect.Ptr {
		log.Fatalf("Do not use pointer for registering a job '%s'\n", jobType)
	}
	// Prevent from panic due to the fact that container name s not in the list
	if _, ok := m.workers[containerName]; !ok {
		w := newWorker(&workerConfig{})
		m.workers[containerName] = &w
	}
	m.workers[containerName].jobTypes[jobType] = j
}

func (m *manager) GetJobTypes() (mm map[string][]string) {
	mm = make(map[string][]string)
	for t, w := range m.workers {
		for typ, _ := range w.jobTypes {
			mm[t] = append(mm[t], typ)
		}
	}
	return
}

func (c *ContainerConfig) validate() (err error) {
	// TODO only contain a-z, A-Z, -, _   unique name
	if c.Name == "" {
		return errors.New("Container config - name cannot be empty")
	}
	if c.Provider == "" {
		return fmt.Errorf("Container config - '%s' provider cannot be empty", c.Name)
	}
	if c.Endpoint == "" {
		return fmt.Errorf("Container config - '%s' endpoint cannot be empty", c.Name)
	}
	if c.Source == "" {
		return fmt.Errorf("Container config - '%s' source cannot be empty", c.Name)
	}
	if c.Concurrency == 0 {
		return fmt.Errorf("Container config - '%s' concurrency cannot be 0", c.Name)
	}
	return
}

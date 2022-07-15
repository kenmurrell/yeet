package workers

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/go-color"
	"gopkg.in/yaml.v3"
)

type ProgramConfig struct {
	MasterBranch string `yaml:"masterbranch"`
	FCRemote     string `yaml:"fcr"`
	RepoDir      string `yaml:"repodir"`
}

type WorkFlowResult struct {
	RepoName string
	Status   Status
	Message  string
}

type Status struct {
	text  string
	color string
	Code  int
}

func (s *Status) ToString() string {
	return s.color + s.text + color.Reset
}

var PASSED Status = Status{"PASSED", color.Green, 0}
var FAILED Status = Status{"FAILED", color.Red, 1}
var CNFLCT Status = Status{"CNFLCT", color.Yellow, 2}
var CURRNT Status = Status{"CURRNT", color.Green, 3}
var BEHIND Status = Status{"BEHIND", color.Yellow, 4}

var config *ProgramConfig
var repolist *RepoList
var GOMAXPROCS int = 4

var RepolistFilename string = "repolist.json"

func loadconfig() *ProgramConfig {
	ex, _ := os.Executable()
	configPath := filepath.Join(filepath.Dir(ex), "config.yaml")
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln("The config.yaml file is missing!", err)
	}
	var tempconfig ProgramConfig
	err = yaml.Unmarshal(yamlFile, &tempconfig)
	if err != nil {
		panic(err)
	}
	if tempconfig.FCRemote == "" || tempconfig.RepoDir == "" || tempconfig.MasterBranch == "" {
		log.Fatalln("The config.yaml is missing values!")
	}
	return &tempconfig
}

func SetupCmd() {
	config = loadconfig()
	ex, _ := os.Executable()
	repoListPath := filepath.Join(filepath.Dir(ex), RepolistFilename)
	sw := SoloWorker{repoListPath, config.RepoDir}
	r, err := sw.GetList()
	if err != nil {
		msg := fmt.Sprintf("Error loading %s, you may need to run `yeet refresh` first?", RepolistFilename)
		log.Fatalln(msg)
	}
	repolist = r
}

func RefreshCmd() {
	config = loadconfig()
	ex, _ := os.Executable()
	repoListPath := filepath.Join(filepath.Dir(ex), RepolistFilename)
	sw := SoloWorker{repoListPath, config.RepoDir}
	err := sw.Refresh()
	if err != nil {
		panic(err)
	}
	r, _ := sw.GetList()
	n := len(r.RepoList)
	fmt.Printf("Loaded %d repositories into %s.", n, repoListPath)
}

func TakeCmd(target string) {
	runtime.GOMAXPROCS(GOMAXPROCS)
	numCPUs := strconv.Itoa(GOMAXPROCS)
	fmt.Printf("Checking out any %s branches using %s CPUs...\n", color.InYellow(target), color.InYellow(numCPUs))
	start := time.Now()
	done := make(chan *WorkFlowResult)
	defer close(done)
	var n int = len(repolist.RepoList)
	for _, r := range repolist.RepoList {
		init := &RepoWorkerInitializer{r}
		go takeWorkflow(target, init, done)
	}
	fmt.Printf("Started %d processes...\n", n)

	for i := 0; i < n; i++ {
		result := <-done
		fmt.Print(result.Format())
	}

	elapsed := time.Since(start)
	fmt.Printf("Done, took %s", elapsed)
}

func StatusCmd() {
	runtime.GOMAXPROCS(GOMAXPROCS)
	numCPUs := strconv.Itoa(GOMAXPROCS)
	fmt.Printf("Checking hashes using %s CPUs...\n", color.InYellow(numCPUs))
	start := time.Now()
	done := make(chan *WorkFlowResult)
	defer close(done)
	var n int = len(repolist.RepoList)
	for _, r := range repolist.RepoList {
		init := &RepoWorkerInitializer{r}
		go statusWorkflow(init, done)
	}
	fmt.Printf("Started %d processes...\n", n)

	for i := 0; i < n; i++ {
		result := <-done
		fmt.Print(result.Format())
	}

	elapsed := time.Since(start)
	fmt.Printf("Done, took %s", elapsed)
}

func (r *WorkFlowResult) Format() string {
	var filler strings.Builder
	var fillerlen = 31 - len(r.Message)
	for i := 0; i < fillerlen; i++ {
		filler.WriteString(" ")
	}

	return fmt.Sprintf(" %s %s%s%s\n", r.Status.ToString(), r.Message, filler.String(), r.RepoName)
}

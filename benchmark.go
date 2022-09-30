package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

func cleanup(terminusdb_dir string, perf_dir string) {
	os.RemoveAll(terminusdb_dir)
	os.RemoveAll(perf_dir)
}

func run_terminusdb_docker(terminusdb_name string, terminusdb_dir string) {
	// Run Docker Build
	docker_build := exec.Command("docker", "buildx", "build", ".", "--tag", terminusdb_name)
	docker_build.Dir = terminusdb_dir
	_, err := docker_build.Output()
	if err != nil {
		fmt.Printf("Error building Docker: %s\n", err)
		return
	}
	// Run Docker Run
	_, err = exec.Command("docker", "run", "--rm", "--detach", "--name", terminusdb_name, "-p", "6363:6363", terminusdb_name).Output()
	if err != nil {
		fmt.Printf("Error running Docker: %s\n", err)
		return
	}
	// The sleep is needed to give TDB a little bit of time to startup
	time.Sleep(5 * time.Second)
}

func execute_benchmark(commit_hash string) {
	// clone https://github.com/terminusdb-labs/terminusdb-http-perf make dir name with commit
	terminusdb_name := "terminusdb_" + commit_hash
	terminusdb_dir := "/tmp/" + terminusdb_name
	perf_dir := "/tmp/perf_" + commit_hash
	_, err := exec.Command("git", "clone", "https://github.com/terminusdb-labs/terminusdb-http-perf.git", perf_dir).Output()
	if err != nil {
		fmt.Printf("Error cloning: %s\n", err)
		return
	}
	_, err = exec.Command("git", "clone", "https://github.com/terminusdb/terminusdb.git", terminusdb_dir).Output()
	if err != nil {
		fmt.Printf("Error cloning: %s\n", err)
		return
	}
	checkout_cmd := exec.Command("git", "checkout", commit_hash)
	checkout_cmd.Dir = terminusdb_dir
	_, err = checkout_cmd.Output()
	if err != nil {
		fmt.Printf("Error checking out: %s\n", err)
		return
	}
	run_terminusdb_docker(terminusdb_name, terminusdb_dir)
	// Run the k6 tests
	perf_json_output := perf_dir + "/output.json"
	_, err = exec.Command("k6", "run", "--no-summary", "--no-usage-report", "--iterations", "10", "--out", "json="+perf_json_output, perf_dir+"/response/all.js").Output()
	if err != nil {
		log.Fatal(err)
		return
	}
	json, _ := ioutil.ReadFile(perf_json_output)
	fmt.Print(string(json))
	// Stop docker image
	_, err = exec.Command("docker", "stop", terminusdb_name).Output()
	if err != nil {
		fmt.Printf("Error stopping Docker container: %s\n", err)
		return
	}
	// Remove docker image and cloned directories
	_, err = exec.Command("docker", "rmi", "--force", terminusdb_name).Output()
	if err != nil {
		fmt.Printf("Error deleting Docker image: %s\n", err)
		return
	}
	cleanup(terminusdb_dir, perf_dir)
}

func main() {
	commit_hash := os.Args[1]
	execute_benchmark(commit_hash)
}

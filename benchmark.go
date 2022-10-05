package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

type BenchmarkConfig struct {
	LegoDemoFolder  string `json:"lego_demo_folder"`
	BenchmarkFolder string `json:"benchmark_output_folder"`
}

func cleanup(terminusdb_name string, terminusdb_dir string) {
	os.RemoveAll(terminusdb_dir)
	// Remove docker image and cloned directories
	_, err := exec.Command("docker", "rmi", "--force", terminusdb_name).Output()
	if err != nil {
		fmt.Printf("Error deleting Docker image: %s\n", err)
		return
	}
}

func clone_and_build_terminusdb(commit_hash string, terminusdb_name string, terminusdb_dir string) {
	_, err := exec.Command("git", "clone", "https://github.com/terminusdb/terminusdb.git", terminusdb_dir).Output()
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
	// Run Docker Build
	docker_build := exec.Command("docker", "buildx", "build", ".", "--tag", terminusdb_name)
	docker_build.Dir = terminusdb_dir
	var stdout []byte
	stdout, err = docker_build.Output()
	fmt.Printf(string(stdout))
	if err != nil {
		fmt.Printf("Error building Docker: %s\n", err)
		return
	}
}

func run_terminusdb_docker(terminusdb_name string, demo_data_folder string) {
	// Run Docker Run
	stdout, err := exec.Command("docker", "run", "--rm", "--detach",
		"-v", demo_data_folder+":/app/demo_data",
		"--name", terminusdb_name, "-p", "6363:6363", terminusdb_name).Output()
	if err != nil {
		fmt.Println(string(stdout))
		fmt.Printf("Error running Docker: %s\n", err)
		return
	}
	// The sleep is needed to give TDB a little bit of time to startup
	time.Sleep(5 * time.Second)
}

func stop_terminusdb_docker(terminusdb_name string) bool {
	_, err := exec.Command("docker", "stop", terminusdb_name).Output()
	if err != nil {
		fmt.Printf("Error stopping Docker container: %s\n", err)
		return false
	}
	time.Sleep(30 * time.Second)
	return true
}

func execute_js_benchmark(terminusdb_name string, terminusdb_dir string, config BenchmarkConfig) {
	fmt.Println("[JS BENCHMARK]")
	run_terminusdb_docker(terminusdb_name, config.LegoDemoFolder)
	test_dir := fmt.Sprintf("%s/tests", terminusdb_dir)
	npm_ci_command := exec.Command("npm", "ci")
	npm_ci_command.Dir = test_dir
	_, _ = npm_ci_command.Output()
	npm_bench_command := exec.Command("node", "bench.js", "--json")
	npm_bench_command.Dir = test_dir
	stdout, _ := npm_bench_command.Output()
	ioutil.WriteFile(fmt.Sprintf("%s/js_benchmark_%s.json", config.BenchmarkFolder, terminusdb_name), stdout, 0644)
	stop_terminusdb_docker(terminusdb_name)
}

func execute_lego_benchmark(terminusdb_name string, config BenchmarkConfig) {
	fmt.Println("[LEGO BENCHMARK]")
	run_terminusdb_docker(terminusdb_name, config.LegoDemoFolder)
	// Create DB and init schema
	_, _ = exec.Command("docker", "exec", terminusdb_name, "./terminusdb", "db", "create", "admin/lego").Output()
	fmt.Println("Lego DB created")
	_, _ = exec.Command("docker", "exec", terminusdb_name, "bash", "-c", "./terminusdb doc insert admin/lego -f -g schema < /app/demo_data/schema.json").Output()
	fmt.Println("Schema inserted")
	timejson_output := config.BenchmarkFolder + "/lego_" + terminusdb_name + ".json"
	_, stderr := exec.Command("timejson", timejson_output, "docker", "exec", terminusdb_name, "bash", "-c", "./terminusdb doc insert admin/lego < /app/demo_data/objs.json").Output()
	fmt.Println(stderr)
	fmt.Println("Timejson finished")
	stop_terminusdb_docker(terminusdb_name)
}

func execute_k6_benchmark(terminusdb_name string, config BenchmarkConfig) error {
	fmt.Println("[K6 BENCHMARK]")
	perf_dir := "/tmp/perf_" + terminusdb_name
	// clone https://github.com/terminusdb-labs/terminusdb-http-perf make dir name with commit
	_, err := exec.Command("git", "clone", "https://github.com/terminusdb-labs/terminusdb-http-perf.git", perf_dir).Output()
	if err != nil {
		fmt.Printf("Error cloning: %s\n", err)
		return err
	}
	run_terminusdb_docker(terminusdb_name, config.LegoDemoFolder)
	// Run the k6 tests
	perf_json_output := config.BenchmarkFolder + "/k6_output.json"
	_, err = exec.Command("k6", "run", "--no-summary", "--no-usage-report", "--iterations", "10", "--out", "json="+perf_json_output, perf_dir+"/response/all.js").Output()
	if err != nil {
		log.Fatal(err)
		return err
	}
	// Stop docker image
	stopped := stop_terminusdb_docker(terminusdb_name)
	if !stopped {
		log.Fatal("Could not stop Docker container")
		return errors.New("Could not stop Docker container")
	}
	// Remove k6 performance benchmark dir
	os.RemoveAll(perf_dir)
	return err

}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage %s [commit_id] [benchmark_type]\n", os.Args[0])
		os.Exit(1)
	}
	commit_hash := os.Args[1]
	benchmark_type := os.Args[2]
	terminusdb_name := "terminusdb_" + commit_hash
	terminusdb_dir := "/tmp/" + terminusdb_name
	var config BenchmarkConfig
	dirname, _ := os.UserHomeDir()
	raw_json, _ := ioutil.ReadFile(dirname + "/.tdb_benchmark_config.json")
	fmt.Println(dirname + "/.tdb_benchmark_config.json")
	err := json.Unmarshal(raw_json, &config)
	if err != nil {
		log.Fatal("Could not read config json")
		fmt.Println(err)
		os.Exit(3)
	}
	clone_and_build_terminusdb(commit_hash, terminusdb_name, terminusdb_dir)
	switch benchmark_type {
	case "k6":
		execute_k6_benchmark(terminusdb_name, config)
	case "lego":
		execute_lego_benchmark(terminusdb_name, config)
	case "js":
		execute_js_benchmark(terminusdb_name, terminusdb_dir, config)
	default:
		execute_js_benchmark(terminusdb_name, terminusdb_dir, config)
		execute_k6_benchmark(terminusdb_name, config)
		execute_lego_benchmark(terminusdb_name, config)
	}
	cleanup(terminusdb_name, terminusdb_dir)
}

/*
© Copyright IBM Corporation 2017

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// runmqserver initializes, creates and starts a queue manager, as PID 1 in a container
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// resolveLicenseFile returns the file name of the MQ license file, taking into
// account the language set by the LANG environment variable
func resolveLicenseFile() string {
	lang, ok := os.LookupEnv("LANG")
	if !ok {
		return "English.txt"
	}
	switch {
	case strings.HasPrefix(lang, "zh_TW"):
		return "Chinese_TW.txt"
	case strings.HasPrefix(lang, "zh"):
		return "Chinese.txt"
	case strings.HasPrefix(lang, "cs"):
		return "Czech.txt"
	case strings.HasPrefix(lang, "fr"):
		return "French.txt"
	case strings.HasPrefix(lang, "de"):
		return "German.txt"
	case strings.HasPrefix(lang, "el"):
		return "Greek.txt"
	case strings.HasPrefix(lang, "id"):
		return "Indonesian.txt"
	case strings.HasPrefix(lang, "it"):
		return "Italian.txt"
	case strings.HasPrefix(lang, "ja"):
		return "Japanese.txt"
	case strings.HasPrefix(lang, "ko"):
		return "Korean.txt"
	case strings.HasPrefix(lang, "lt"):
		return "Lithuanian.txt"
	case strings.HasPrefix(lang, "pl"):
		return "Polish.txt"
	case strings.HasPrefix(lang, "pt"):
		return "Portugese.txt"
	case strings.HasPrefix(lang, "ru"):
		return "Russian.txt"
	case strings.HasPrefix(lang, "sl"):
		return "Slovenian.txt"
	case strings.HasPrefix(lang, "es"):
		return "Spanish.txt"
	case strings.HasPrefix(lang, "tr"):
		return "Turkish.txt"
	}
	return "English.txt"
}

func checkLicense() {
	lic, ok := os.LookupEnv("LICENSE")
	switch {
	case ok && lic == "accept":
		return
	case ok && lic == "view":
		file := filepath.Join("/opt/mqm/licenses", resolveLicenseFile())
		buf, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(buf))
		os.Exit(1)
	}
	fmt.Println("Error: Set environment variable LICENSE=accept to indicate acceptance of license terms and conditions.")
	fmt.Println("License agreements and information can be viewed by setting the environment variable LICENSE=view.  You can also set the LANG environment variable to view the license in a different language.")
	os.Exit(1)
}

// sanitizeQueueManagerName removes any invalid characters from a queue manager name
func sanitizeQueueManagerName(name string) string {
	var re = regexp.MustCompile("[^a-zA-Z0-9._%/]")
	return re.ReplaceAllString(name, "")
}

// GetQueueManagerName resolves the queue manager name to use.  Resolved from
// either an environment variable, or the hostname.
func getQueueManagerName() (string, error) {
	var name string
	var err error
	name, ok := os.LookupEnv("MQ_QMGR_NAME")
	if !ok || name == "" {
		name, err = os.Hostname()
		if err != nil {
			return "", err
		}
		name = sanitizeQueueManagerName(name)
	}
	// TODO: What if the specified env variable is an invalid name?
	return name, nil
}

// runCommand runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
func runCommand(name string, arg ...string) (string, int, error) {
	cmd := exec.Command(name, arg...)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		var rc int
		// Only works on Linux
		if runtime.GOOS == "linux" {
			var ws unix.WaitStatus
			unix.Wait4(cmd.Process.Pid, &ws, 0, nil)
			rc = ws.ExitStatus()
		} else {
			rc = -1
		}
		if rc == 0 {
			return string(out), rc, nil
		}
		return string(out), rc, err
	}
	return string(out), 0, nil
}

// createDirStructure creates the default MQ directory structure under /var/mqm
func createDirStructure() {
	out, _, err := runCommand("/opt/mqm/bin/crtmqdir", "-f", "-s")
	if err != nil {
		log.Fatalf("Error creating directory structure: %v\n", string(out))
	}
	log.Println("Created directory structure under /var/mqm")
}

func createQueueManager(name string) {
	log.Printf("Creating queue manager %v", name)
	out, rc, err := runCommand("crtmqm", "-q", "-p", "1414", name)
	if err != nil {
		// 8=Queue manager exists, which is fine
		if rc != 8 {
			log.Printf("crtmqm returned %v", rc)
			log.Fatalln(string(out))
		} else {
			log.Printf("Detected existing queue manager %v", name)
			return
		}
	}
}

func updateCommandLevel() {
	level, ok := os.LookupEnv("MQ_CMDLEVEL")
	if ok && level != "" {
		out, rc, err := runCommand("strmqm", "-e", "CMDLEVEL="+level)
		if err != nil {
			log.Fatalf("Error %v setting CMDLEVEL: %v", rc, string(out))
		}
	}
}

func startQueueManager() {
	log.Println("Starting queue manager")
	out, rc, err := runCommand("strmqm")
	if err != nil {
		log.Fatalf("Error %v starting queue manager: %v", rc, string(out))
	}
	log.Println("Started queue manager")
}

func configureQueueManager() {
	const configDir string = "/etc/mqm"
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".mqsc") {
			abs := filepath.Join(configDir, file.Name())
			mqsc, err := ioutil.ReadFile(abs)
			if err != nil {
				log.Fatal(err)
			}
			cmd := exec.Command("runmqsc")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}
			stdin.Write(mqsc)
			stdin.Close()
			// Run the command and wait for completion
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Println(err)
			}
			// Print the runmqsc output, adding tab characters to make it more readable as part of the log
			log.Printf("Output for \"runmqsc\" with %v:\n\t%v", abs, strings.Replace(string(out), "\n", "\n\t", -1))
		}
	}
}

func stopQueueManager() {
	log.Println("Stopping queue manager")
	out, _, err := runCommand("endmqm", "-w")
	if err != nil {
		log.Fatalf("Error stopping queue manager: %v", string(out))
	}
	log.Println("Stopped queue manager")
}

// createTerminateChannel creates a channel which will be closed when SIGTERM
// is received.
func createTerminateChannel() chan struct{} {
	done := make(chan struct{})
	// Handle SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-c
		log.Printf("Signal received: %v", sig)
		stopQueueManager()
		close(done)
	}()
	return done
}

// createReaperChannel creates a channel which will be used to reap zombie
// (defunct) processes.  This is a responsibility of processes running
// as PID 1.
func createReaper() {
	// Handle SIGCHLD
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGCHLD)
	go func() {
		for {
			<-c
			for {
				var ws unix.WaitStatus
				_, err := unix.Wait4(-1, &ws, 0, nil)
				// If err indicates "no child processes" left to reap, then
				// wait for next SIGCHLD signal
				if err == unix.ECHILD {
					break
				}
			}
		}
	}()
}

func main() {
	createReaper()
	checkLicense()
	// Start SIGTERM handler channel
	done := createTerminateChannel()

	name, err := getQueueManagerName()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Using queue manager name: %v", name)

	logConfig()
	err = createVolume("/mnt/mqm")
	if err != nil {
		log.Fatal(err)
	}
	createDirStructure()
	createQueueManager(name)
	updateCommandLevel()
	startQueueManager()
	configureQueueManager()
	// Wait for terminate signal
	<-done
}

/*
Author: Sriram Kaushik
Purpose: Slave worker will listen on a chosen port for any requests from the master.
Once a request comes, it will handle it (and run the ansible playbook to deploy splunk-uf for that request env)
slaves can run on multiple VMs or ports.
*/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/kaushiksriram100/ansible-deployer-dist/shyunutils"
)

const OO_API_TOKEN = "<sorry>"

func RunAnsible(logdir *string, logfile *os.File, request *shyunutils.RequestMessage, subresponsestream chan string) {
	log.SetOutput(logfile)

	var playbook_action, playbook_path, jar_path, assembly_file string = (*request).PlaybookAction, (*request).PlaybookPath, (*request).InvJar, (*request).Nspath_file

	OO_ORG, OO_ASSEMBLY, OO_ENV, OO_PLATFORM := (*request).OO_Org, (*request).OO_Assembly, (*request).OO_Env, (*request).OO_Platform

	//45 minute timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3100*time.Second)
	defer cancel()

	playbook_to_run := (playbook_path) + "/" + "tasks/" + (playbook_action)

	//we need to modify the OO_PLATFORM as the oneops jar expects -l arguement as platform-<actual platformname>-compute.
	L_OO_PLATFORM := "platform-" + OO_PLATFORM + "-compute"
	//fmt.Println(OO_API_TOKEN, OO_ORG, OO_ENV, OO_ASSEMBLY)
	cmd := exec.CommandContext(ctx, "ansible-playbook", "-l", L_OO_PLATFORM, "--user=app", "-i", jar_path, playbook_to_run, "--extra-vars", "OO_ORG="+OO_ORG+" OO_ASSEMBLY="+OO_ASSEMBLY+" OO_PLATFORM="+OO_PLATFORM+" OO_ENV="+OO_ENV+"")
	env := os.Environ()

	env = append(env, fmt.Sprintf("OO_API_TOKEN=%s", OO_API_TOKEN), fmt.Sprintf("OO_ORG=%s", OO_ORG), fmt.Sprintf("OO_ASSEMBLY=%s", OO_ASSEMBLY), fmt.Sprintf("OO_ENV=%s", OO_ENV), fmt.Sprintf("OO_ENDPOINT=%s", "https://oneops.prod.walmart.com/"), fmt.Sprintf("ANSIBLE_HOST_KEY_CHECKING=%s", "False"))
	cmd.Env = env

	//Above note that you can simply add env variables and skip passing --extra-vars in playbook. But to to that you should modify the playbook/template to do lookup('env', 'OO_ENV') instead of checking to the variable directly. Not tested.

	// create a output file.
	//fmt.Println(assembly_file)
	outfile, err := os.Create((*logdir) + "/" + assembly_file + ".output")
	if err != nil {
		log.Fatal("Error: Unable to Create output file but I will still proceed")
	}

	defer outfile.Close()

	cmd.Stdout = outfile
	cmd.Stderr = outfile

	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		subresponsestream <- `ERROR: Context deadline exceeded. Ansible is taking more than 50 mins to complete. I Killed ansible process and cleaned all resources. May be network latency or organic growth. if organic growth then consider increasing context timeout for:` + OO_ORG + "_" + OO_ASSEMBLY + "_" + OO_PLATFORM + "_" + OO_ENV
		runtime.Goexit()
	}

	//If there was not context deadline exceeded, then there is some issue in playbook and exited with non 0 exit code.
	if err != nil {
		subresponsestream <- `WARNING: Playbook failed on some or all hosts with exit code not equal 0. Check ansible output logs for:` + OO_ORG + "_" + OO_ASSEMBLY + "_" + OO_PLATFORM + "_" + OO_ENV
		runtime.Goexit()
	}
	subresponsestream <- "INFO: Completed deployment with no major failures. Ok to proceed"
	runtime.Goexit()

}

func HandleTCPConnection(conn net.Conn, logfile *os.File, log_file_path *string) {

	log.SetOutput(logfile)

	log.Print("INFO:Handling request...")

	//FIX->Instead of Defer, you can close it after decoding(before RunAnsible), so that connection is not blocked due to ansible runs
	defer func() {
		log.Print("INFO:Closing Connection...")
		conn.Close()
	}()

	//timeoutDuration := 10 * time.Second
	//directly read from the conn
	requestjson := json.NewDecoder(conn)

	//Read data, data is in []bytes.. WORK FROM HERE.

	requestmessage := &shyunutils.RequestMessage{}

	//decode the data from conn. under the hoods, unmarshall is what happens with decode.

	err := requestjson.Decode(requestmessage)

	if err != nil {
		log.Print("ERROR: Unable to parse the request")
		runtime.Goexit()
	}

	//now that we have all data in the []byte, lets send this to a function from where we will trigger anisble jobs
	subresponsestream := make(chan string)

	//Go routine inside goroutine.. I am improving skills here..
	go RunAnsible(log_file_path, logfile, requestmessage, subresponsestream)

	log.Print(<-subresponsestream)
	runtime.Goexit()

	//compile the subresponsestream and send to responsestream for main.

	//	log.Print(string(request), totalread, err) //comment this later

}

func main() {

	//collect arguments passed
	port := flag.String("port", "3001", "port to listen for tcp default to 3001")
	log_file_path := flag.String("logfilepath", "/tmp", "-logfile <> default /tmp")

	flag.Parse()

	//Create log file. 1.create the director if not exists, then create the log file.
	logfile, err := shyunutils.CreateLogFile(log_file_path, "shyunslave")

	if err != nil {
		fmt.Println("Error Occured while creating log file:", err)
		return
	}
	log.SetOutput(logfile)
	defer logfile.Close()

	/*Create a listener that will listen for connections on a tcp port.*/

	//listen on the port for TCP Connection

	listener, err := net.Listen("tcp", ":"+(*port))

	if err != nil {
		log.Fatal("ERROR: I am unable to start listening. Something fundamentally wrong with ethos..")
		return
	}

	//start a for loop to start listening to TCP and send a received connection to HandleConnection function

	log.Print("INFO: Ready to Accepting ")
	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Fatal("ERROR: Listener missed a request")
			continue
		}

		go HandleTCPConnection(conn, logfile, log_file_path)

	}

}

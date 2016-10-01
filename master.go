package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"github.com/unixpickle/jobempire/jobadmin"
	"github.com/unixpickle/jobempire/jobproto"
)

func MasterMain(slavePort, adminPort int, slavePass, adminPass string, jobFile string) {
	jobs, err := readJobs(jobFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read jobs:", err)
		os.Exit(1)
	}

	slaveListener, err := net.Listen("tcp", ":"+strconv.Itoa(slavePort))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to listen for slaves:", err)
		os.Exit(1)
	}

	adminListener, err := net.Listen("tcp", ":"+strconv.Itoa(adminPort))
	if err != nil {
		slaveListener.Close()
		fmt.Fprintln(os.Stderr, "Failed to listen for admins:", err)
		os.Exit(1)
	}

	log.Println("Listening on ports", slavePort, "and", adminPort)

	defer slaveListener.Close()
	defer adminListener.Close()

	handler := &MasterHandler{
		Scheduler: jobadmin.NewScheduler(),
		Auth:      NewMasterAuth(adminPass),
		Templates: parseTemplates(),
		JobsPath:  jobFile,
	}
	handler.Scheduler.SetJobs(jobs)

	go http.Serve(adminListener, handler)
	go func() {
		for {
			conn, err := slaveListener.Accept()
			if err != nil {
				return
			}
			if handler.Scheduler.Terminated() {
				conn.Close()
				return
			}
			go func() {
				master, err := jobproto.NewMasterConnAuth(conn, slavePass)
				if err != nil {
					log.Println("Slave", conn.RemoteAddr(), "failed to authenticate.")
					return
				}
				log.Println("Slave", conn.RemoteAddr(), "successfully joined.")
				handler.Scheduler.AddMaster(jobadmin.RunLiveMaster(master), false)
			}()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down...")

	handler.Scheduler.Terminate()
	handler.Scheduler.Wait(nil)
}

func readJobs(file string) ([]*jobadmin.Job, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, nil
	}
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var jobs []*jobadmin.Job
	if err := json.Unmarshal(contents, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func parseTemplates() *template.Template {
	files := []string{}
	for _, n := range AssetNames() {
		if strings.HasSuffix(n, ".html") {
			files = append(files, n)
		}
	}
	var body bytes.Buffer
	for _, f := range files {
		data, err := Asset(f)
		if err != nil {
			panic(err)
		}
		body.Write(data)
	}
	res := template.New("master")
	res.Funcs(template.FuncMap{
		"masters":      templateMasters,
		"pair":         templatePair,
		"reverse":      templateReverse,
		"jsonPass":     templateJSONPass,
		"reverseIndex": templateReverseIndex,
	})
	return template.Must(res.Parse(body.String()))
}

type masterAutoPair struct {
	Master *jobadmin.LiveMaster
	Auto   bool
}

func templateMasters(s *jobadmin.Scheduler) ([]masterAutoPair, error) {
	masters, auto, err := s.Masters()
	if err != nil {
		return nil, err
	}
	pairs := make([]masterAutoPair, len(masters))
	for i, m := range masters {
		pairs[i] = masterAutoPair{m, auto[i]}
	}
	return pairs, nil
}

func templatePair(x, y interface{}) []interface{} {
	return []interface{}{x, y}
}

func templateReverse(x interface{}) (interface{}, error) {
	oldVal := reflect.ValueOf(x)
	if oldVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("reverse: expected slice but got %T", x)
	}

	slice := reflect.MakeSlice(reflect.TypeOf(x), oldVal.Len(), oldVal.Len())
	reflect.Copy(slice, oldVal)
	for i := 0; i < slice.Len()/2; i++ {
		slot1 := slice.Index(i)
		slot2 := slice.Index(slice.Len() - (i + 1))
		val1 := reflect.ValueOf(slot1.Interface())
		val2 := reflect.ValueOf(slot2.Interface())
		slot1.Set(val2)
		slot2.Set(val1)
	}
	return slice.Interface(), nil
}

func templateJSONPass(x interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(x)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	err = json.Unmarshal(data, &res)
	return res, err
}

func templateReverseIndex(i, count int) int {
	return count - (i + 1)
}

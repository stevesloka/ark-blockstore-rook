package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	argListenPort = flag.Int("listen-port", 9080, "port to have API listen")
)

// /block/eu-west-2/fbn-prod-k8s-backup-primary/pvcs/0b3f70b1-f7fd-4c07-a5d2-7eb107d26fa3/replicapool/pvc-faacb401-4f9b-11e8-b3de-0a7763bb1b08
// (POST "/block/{region}/{bucket}/{prefix}/{snapid}/{pool}/{imagename}")
func createBlockRoute(w http.ResponseWriter, r *http.Request) {
	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapid := vars["snapid"]
	imageName := vars["imagename"]
	region := vars["region"]
	bucket := vars["bucket"]
	prefix := vars["prefix"]

	fmt.Printf("poolname: %v\nsnapshotname: %v\nimageName: %v\nregion: %v \nbucket: %v\nprefix: %v\n", poolName, snapid, imageName, region, bucket, prefix)

	backupfile := fmt.Sprintf("/tmp/backup/%s/%s/%s", snapid, poolName, imageName)

	// $ aws cp s3://{bucket}/{prefix}/{snap-name}/{pool-name} /tmp/backup/{snapid}/{pool}/{imagename} --sse --region {region}
	cmd := "aws"
	args := []string{"s3", "cp", fmt.Sprintf("s3://%s/%s/%s/%s/%s", bucket, prefix, snapid, poolName, imageName), backupfile, "--sse"}
	fmt.Printf("%s %v\n", cmd, args)

	if out, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error creating the volume (aws): ", err)
		fmt.Fprintln(os.Stderr, "command output: ", string(out))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// $ rbd import /tmp/backup/{snapshotName}/{pool}/{image} {pool}/{image} --export-format 2
	// , "--image-feature", "layering"
	cmd = "rbd"
	args = []string{"import", "--export-format", "2", backupfile, fmt.Sprintf("%s/%s", poolName, imageName)}
	fmt.Printf("%s %v\n", cmd, args)

	if out, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error creating the volume (rbd): ", err)
		fmt.Fprintln(os.Stderr, "command output: ", string(out))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// (POST "/snapshot/{region}/{bucket}/{prefix}/{snapname}/{pool}/{imagename}")
func createSnapshotRoute(w http.ResponseWriter, r *http.Request) {
	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapshotName := vars["snapname"]
	imageName := vars["imagename"]
	region := vars["region"]
	bucket := vars["bucket"]
	prefix := vars["prefix"]

	fmt.Printf("poolname: %v\nsnapshotname: %v\nimageName: %v\nregion: %v \nbucket: %v\nprefix: %v\n", poolName, snapshotName, imageName, region, bucket, prefix)

	os.MkdirAll(fmt.Sprintf("/tmp/backup/%s/%s", snapshotName, poolName), os.FileMode(0522))

	backupfile := fmt.Sprintf("/tmp/backup/%s/%s/%s", snapshotName, poolName, imageName)
	// $ rbd export {pool}/{image} /tmp/backup/{snapshotName}/{pool}/{image}
	cmd := "rbd"
	args := []string{"export", "--export-format", "2", fmt.Sprintf("%s/%s", poolName, imageName), backupfile}

	fmt.Printf("%s %v\n", cmd, args)

	if out, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error creating snapshot (rbd): ", err)
		fmt.Fprintln(os.Stderr, "command output: ", string(out))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// $ aws mv {backupfile} s3://{bucket}/{prefix}/{pool-name}/{snap-name}/{imageNmae} --sse --region {region}
	cmd = "aws"
	args = []string{"s3", "mv", backupfile, fmt.Sprintf("s3://%s/%s/%s/%s/%s", bucket, prefix, snapshotName, poolName, imageName), "--sse"}
	fmt.Printf("%s %v\n", cmd, args)

	if out, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error creating snapshot (aws): ", err)
		fmt.Fprintln(os.Stderr, "command output: ", string(out))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// (DELETE "/snapshot/{region}/{bucket}/{prefix}/{pool}/{snapname}")
func deleteSnapshotRoute(w http.ResponseWriter, r *http.Request) {

	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapshotName := vars["snapname"]
	region := vars["region"]
	bucket := vars["bucket"]
	prefix := vars["prefix"]

	fmt.Printf("poolname: %v\nsnapshotname: %v\nregion: %v \nbucket: %v\nprefix: %v\n", poolName, snapshotName, region, bucket, prefix)

	// $ aws s3 rm s3://{bucket}/{prefix}/{snap-name}/{pool-name} --sse --region {region}
	cmd := "aws"
	args := []string{"s3", "rm", fmt.Sprintf("s3://%s/%s/%s/%s", bucket, prefix, snapshotName, poolName), "--sse"}
	fmt.Printf("%s %v\n", cmd, args)
	if err := exec.Command(cmd, args...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		fmt.Println("Successfully deleted snapshot")
	}
}

// (GET "/snapshot/{region}/{bucket}/{prefix}/{pool}/{imagename}")
func listSnapshotsRoute(w http.ResponseWriter, r *http.Request) {

	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	imageName := vars["imagename"]
	// region := vars["region"]
	bucket := vars["bucket"]
	prefix := vars["prefix"]

	cmd := "aws"
	args := []string{"s3", "ls", fmt.Sprintf("s3://%s/%s/%s/%s", bucket, prefix, poolName, imageName), "--sse"}
	fmt.Printf("%s %v\n", cmd, args)
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		fmt.Println("Successfully listed snapshots")
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// (GET "/block/{pool}/{image}")
func getBlockRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolName := vars["pool"]
	imageName := vars["imagename"]

	initRook()

	conn, _ := rados.NewConn()
	conn.ReadDefaultConfigFile()
	conn.Connect()
	context, err := conn.OpenIOContext(poolName)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	image := rbd.GetImage(context, imageName)
	image.Open(context)
	size, err := image.GetSize()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(size); err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// (GET "/block/{pool}")
func listBlocksRoute(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	poolName := vars["pool"]

	initRook()

	conn, _ := rados.NewConn()
	conn.ReadDefaultConfigFile()
	conn.Connect()
	context, err := conn.OpenIOContext(poolName)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	names, err := rbd.GetImageNames(context)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(names); err != nil {
		fmt.Fprintln(os.Stderr, err)
		w.WriteHeader(http.StatusInternalServerError)
	}

}

// initRook sets up the local container for access to ceph pods
func initRook() {
	var (
		cmdOut []byte
		err    error
	)

	cmd := "/usr/local/bin/toolbox.sh"
	args := []string{}
	if cmdOut, err = exec.Command(cmd, args...).CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error init'ing the toolbox: ", string(cmdOut))
	} else {
		log.Println("Called initRook()")
	}
}

// Default (GET "/")
func indexRoute(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s", "welcome to rook-rest-api!")
}

func main() {
	flag.Parse()
	log.Println("rook-rest-api is up and running!", time.Now())

	// Configure router
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", indexRoute)
	router.HandleFunc("/block/{region}/{bucket}/{prefix}/{snapid}/{pool}/{imagename}", createBlockRoute).Methods("POST")
	router.HandleFunc("/snapshot/{region}/{bucket}/{prefix}/{snapname}/{pool}/{imagename}", createSnapshotRoute).Methods("POST")
	router.HandleFunc("/snapshot/{region}/{bucket}/{prefix}/{snapname}/{pool}", deleteSnapshotRoute).Methods("DELETE")
	router.HandleFunc("/snapshot/{region}/{bucket}/{prefix}/{pool}/{imagename}", listSnapshotsRoute).Methods("GET")
	router.HandleFunc("/block/{pool}", listBlocksRoute).Methods("GET")
	router.HandleFunc("/block/{pool}/{imagename}", getBlockRoute).Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	// Start server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *argListenPort), loggedRouter))
}

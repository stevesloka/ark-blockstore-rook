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

// (POST "/block/{pool}/{snapid}/{imagename}")
func createBlockRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapid := vars["snapid"]
	imagename := vars["imagename"]

	err := copyVolumes(poolName, snapid, imagename)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, "There was an error creating the volume: ", err)
	}
}

// (POST "/snapshot/{pool}/{snapname}/{imagename}")
func createSnapshotRoute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapshotName := vars["snapname"]
	imageName := vars["imagename"]

	err := copyVolumes(poolName, imageName, snapshotName)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, "There was an error creating snapshot: ", err)
	}

}

// copyVolumes copies one ceph volume to another
func copyVolumes(poolName, sourceImage, destImage string) error {
	initRook()

	// $ rbd cp {pool}/{image} {pool}/{image}
	cmd := "rbd"
	args := []string{"cp", fmt.Sprintf("%s/%s", poolName, sourceImage), fmt.Sprintf("%s/%s", poolName, destImage)}

	if _, err := exec.Command(cmd, args...).Output(); err != nil {
		return err
	}

	return nil
}

// (DELETE "/snapshot/{pool}/{snapname}")
func deleteSnapshotRoute(w http.ResponseWriter, r *http.Request) {

	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	snapshotName := vars["snapname"]

	// $ rbd --pool {pool-name} snap rm --snap {snap-name} {image-name}
	cmd := "rbd"
	args := []string{"rm", snapshotName, "-p", poolName}
	if err := exec.Command(cmd, args...).Run(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println("Successfully deleted snapshot")
	}
}

// (GET "/snapshot/{pool}/{imagename}")
func listSnapshotsRoute(w http.ResponseWriter, r *http.Request) {

	initRook()

	vars := mux.Vars(r)
	poolName := vars["pool"]
	imageName := vars["imagename"]

	conn, _ := rados.NewConn()
	conn.ReadDefaultConfigFile()
	conn.Connect()
	context, _ := conn.OpenIOContext(poolName)

	image := rbd.GetImage(context, imageName)
	snaps, err := image.GetSnapshotNames()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(snaps); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
		return
	}

	image := rbd.GetImage(context, imageName)
	image.Open(context)
	size, err := image.GetSize()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(size); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
		return
	}

	names, err := rbd.GetImageNames(context)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(names); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(os.Stderr, err)
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
	if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
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
	router.HandleFunc("/block/{pool}/{snapid}/{imagename}", createBlockRoute).Methods("POST")
	router.HandleFunc("/snapshot/{pool}/{snapname}/{imagename}", createSnapshotRoute).Methods("POST")
	router.HandleFunc("/snapshot/{pool}/{snapname}", deleteSnapshotRoute).Methods("DELETE")
	router.HandleFunc("/snapshot/{pool}/{imagename}", listSnapshotsRoute).Methods("GET")
	router.HandleFunc("/block/{pool}", listBlocksRoute).Methods("GET")
	router.HandleFunc("/block/{pool}/{imagename}", getBlockRoute).Methods("GET")

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)

	// Start server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *argListenPort), loggedRouter))
}

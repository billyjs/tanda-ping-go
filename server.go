package main

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"encoding/json"
	"strconv"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// a struct for a DataStore to hold a mongo session and collection
type DataStore struct {
	session* mgo.Session
	collection* mgo.Collection
}

// a struct for a device with a device_id and a list of pings
type Device struct {
	DeviceId string `json: device_id`
	Pings []int64 `json: pings`
}

// remove all devices and pings from the database
func (ds* DataStore) clearData(w http.ResponseWriter, r* http.Request) {
	ds.collection.RemoveAll(nil)
	w.WriteHeader(http.StatusOK)
}

// get all devices in the database
func (ds* DataStore) getDevices(w http.ResponseWriter, r* http.Request) {
	var devices []Device
	err := ds.collection.Find(nil).All(&devices)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	list := make([]string, len(devices))

	for i, device := range devices {
		list[i] = device.DeviceId
	}

	b, _ := json.Marshal(list)

	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// add a ping to the database
func (ds* DataStore) postPing(w http.ResponseWriter, r* http.Request) {
	var device_id string
	vars := mux.Vars(r)
	device_id = vars["device_id"]
	epoch_time := vars["epoch_time"]

	unix, err :=  strconv.ParseInt(epoch_time, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	selector := bson.M{"deviceid": device_id}
	upsert := bson.M{"$addToSet": bson.M{"pings": unix}}

	_, err = ds.collection.Upsert(selector, upsert)
	if err != nil {
		log.Fatal(err)
	}
}

// get ping(s) for device(s) between the range start and end
func (ds* DataStore) getRange(w http.ResponseWriter, r* http.Request) {
	vars := mux.Vars(r)
	device_id := vars["device_id"]
	from := vars["from"]
	to := vars["to"]

	var start int64
	var end int64
	var b []byte

	t1, err := time.Parse("2006-01-02", from)
	if err != nil {
		// could be unix timestamp or bad request
		i, err :=  strconv.ParseInt(from, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
			w.Write(b)
			return
		}
		start = i
	} else {
		start = t1.Unix()
	}


	t2, err := time.Parse("2006-01-02", to)
	if err != nil {
		// could be unix timestamp or bad request
		i, err :=  strconv.ParseInt(to, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
			w.Write(b)
			return
		}
		end = i
	}else {
		end = t2.AddDate(0, 0, 1).Unix()
	}

	b, bad := getPings(ds, device_id, start, end)
	if bad {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Write(b)

}

// get ping(s) for device(s) on a specific date
func (ds* DataStore) getDate(w http.ResponseWriter, r* http.Request) {
	vars := mux.Vars(r)
	device_id := vars["device_id"]
	date := vars["date"]

	var b []byte

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		empty := make([]int64, 0)
		b, _ = json.Marshal(empty)
		w.Write(b)
		return
	}

	start := t.Unix()
	end := t.AddDate(0, 0, 1).Unix()

	b, bad := getPings(ds, device_id, start, end)
	if bad {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Write(b)


}

// get pings for all devices if device_id == "all" or get pings for a specific device
func getPings(ds* DataStore, device_id string, start int64, end int64)([]byte, bool) {
	var b []byte
	if device_id == "all" {
		var devices []Device
		data := map[string][]int64{}

		err := ds.collection.Find(nil).All(&devices)
		if err != nil {
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
			return b, true
		}

		for _, device := range devices {
			pings := getPingsIn(device.Pings, start, end)
			if pings != nil {
				data[device.DeviceId] = getPingsIn(device.Pings, start, end)
			}
		}

		if data != nil {
			b, _ = json.Marshal(data)
		} else {
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
		}
		return b, false

	} else {
		var device Device
		err := ds.collection.Find(bson.M{"deviceid": device_id}).One(&device)
		if err != nil {
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
			return b, true
		}

		pings := getPingsIn(device.Pings, start, end)

		if pings != nil {
			b, _ = json.Marshal(pings)
		} else {
			empty := make([]int64, 0)
			b, _ = json.Marshal(empty)
		}

		return b, false
	}
}

// Get pings that are in the range [start, end)
func getPingsIn(pings []int64, start int64, end int64)(in []int64) {
	for _, ping := range pings {
		if ping >= start && ping < end {
			in = append(in, ping)
		}
	}
	return in
}

func main() {
	// connect to mongodb
	session, err := mgo.Dial("mongodb://localhost")
	if err != nil {
		log.Fatal(err)
	}

	// close connection to mongodb when program finishes
	defer session.Close()

	// connect to database "go-ping" and collection "devices"
	collection := session.DB("go-ping").C("devices")

	// set "deviceid" in "devices" to be unique (this is because it is the id)
	index := mgo.Index{
		Key: []string{"deviceid"},
		Unique: true,
	}
	err = collection.EnsureIndex(index)
	if err != nil {
		log.Fatal(err)
	}

	// create a DataStore holding the session and collection,
	// this is used to perform ops in the database in other functions
	ds := DataStore{session, collection}

	// start a mux router
	router := mux.NewRouter();

	// set router handlers
	router.HandleFunc("/clear_data", ds.clearData).Methods("POST")
	router.HandleFunc("/devices", ds.getDevices).Methods("GET")
	router.HandleFunc("/{device_id}/{date}", ds.getDate).Methods("GET")
	router.HandleFunc("/{device_id}/{from}/{to}", ds.getRange).Methods("GET")
	router.HandleFunc("/{device_id}/{epoch_time}", ds.postPing).Methods("POST")

	// server start listening on port 3000
	fmt.Println("Listening on port 3000!")
	log.Fatal(http.ListenAndServe(":3000", router))
}

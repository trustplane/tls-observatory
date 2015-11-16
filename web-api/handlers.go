package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/streadway/amqp"

	"github.com/mozilla/TLS-Observer/logger"
	pg "github.com/mozilla/TLS-Observer/modules/postgresmodule"
)

func ScanHandler(w http.ResponseWriter, r *http.Request) {

	log := logger.GetLogger()

	log.WithFields(logrus.Fields{
		"form values": r.Form,
		"headers":     r.Header,
	}).Debug("Received request")

	var (
		status int
		err    error
	)

	defer func() {
		if nil != err {
			http.Error(w, err.Error(), status)
		}
	}()

	val, ok := context.GetOk(r, dbKey)

	if !ok {
		log.Error("Could not find db in request context")
		status = http.StatusInternalServerError
		return
	}

	db := val.(*pg.DB)

	db.Ping()

	domain := r.FormValue("target")

	if validateDomain(domain) {

		conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			log.Fatal(err)
		}
		defer ch.Close()

		scan, err := db.NewScan(domain, -1) //no replay

		sID := strconv.FormatInt(scan.ID, 10)

		if err != nil {
			log.WithFields(logrus.Fields{
				"domain": domain,
				"error":  err.Error(),
			}).Error("Could not create new scan")
			status = http.StatusInternalServerError
			return
		}

		resp := fmt.Sprintf(`{"scan_id":"%d"}`, scan.ID)

		_, err = w.Write([]byte(resp))

		if err != nil {
			log.WithFields(logrus.Fields{
				"domain":  domain,
				"error":   err.Error(),
				"scan_id": scan.ID,
			}).Error("Could not write scan id to respons")
			status = http.StatusInternalServerError
			return
		}

		status = http.StatusOK

		log.Println("Publishing ", domain)
		err = ch.Publish(
			"amq.direct", // exchange
			"scan_ready", // routing key
			false,        // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(sID),
			})

	} else {
		status = http.StatusBadRequest
		return
	}

}

func ResultHandler(w http.ResponseWriter, r *http.Request) {

	var (
		status int
		err    error
	)

	defer func() {
		if nil != err {
			http.Error(w, err.Error(), status)
		}
	}()

	domain := r.FormValue("id")

	if validateDomain(domain) {

		status = http.StatusOK

	} else {
		status = http.StatusBadRequest
		return
	}

}

func CertificateHandler(w http.ResponseWriter, r *http.Request) {

	var (
		status int
		err    error
	)

	defer func() {
		if nil != err {
			http.Error(w, err.Error(), status)
		}
	}()

	domain := r.FormValue("target")

	if validateDomain(domain) {

		//		raw := r.FormValue("raw")

		//		rawCert := false

		//		if raw == "true" {
		//			rawCert = true
		//		}

		status = http.StatusOK

	} else {
		status = http.StatusBadRequest
		return
	}

}

func validateDomain(domain string) bool {

	// TODO
	// Need to validate the domain, in a way,
	// before passing it to the retriever queue

	return true
}
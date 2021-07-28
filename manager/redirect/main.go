package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("dev.zippytal.com/", func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw,r,"https://dev.zippytal.com/",http.StatusPermanentRedirect)
	})
	http.HandleFunc("app.zippytal.com/", func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw,r,"https://app.zippytal.com/",http.StatusPermanentRedirect)
	})
	http.HandleFunc("zippytal.com/", func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw,r,"https://zippytal.com/",http.StatusPermanentRedirect)
	})
	log.Fatalln(http.ListenAndServe(":80", nil))
}
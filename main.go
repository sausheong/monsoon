package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/local"
)

var (
	wd          string
	bin         string
	model       string
	promptFile  string
	gpuLayers   string
	threads     string
	contextSize string
)

// initialise to load environment variable from .env file
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	wd, err = os.Getwd()
	if err != nil {
		log.Fatal("Error getting current directory")
	}
	bin = os.Getenv("LOCAL_LLM_BIN")
	model = os.Getenv(("LOCAL_LLM_MODEL"))
	promptFile = os.Getenv(("LOCAL_LLM_PROMPT_FILE"))
	gpuLayers = os.Getenv(("LOCAL_LLM_NUM_GPU_LAYERS"))
	threads = os.Getenv(("LOCAL_LLM_NUM_CPU_CORES"))
	contextSize = os.Getenv(("LOCAL_LLM_CONTEXT"))

}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	r.Get("/", index)
	r.Post("/run", run)
	log.Println("\033[93mMonsoon started. Press CTRL+C to quit.\033[0m")
	http.ListenAndServe(":"+os.Getenv("PORT"), r)
}

// index
func index(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("static/index.html")
	t.Execute(w, nil)
}

// call the LLM and return the response
func run(w http.ResponseWriter, r *http.Request) {
	prompt := struct {
		Input string `json:"input"`
	}{}
	// decode JSON from client
	err := json.NewDecoder(r.Body).Decode(&prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// create the LLM
	bin := fmt.Sprintf("%s/%s", wd, bin)
	args := fmt.Sprintf("-m %s/%s -t %s --temp 0 -c %s"+
		" -ngl %s --file %s/%s -r \"[Question]\" -p",
		wd, model, threads, contextSize, gpuLayers, wd, promptFile)
	llm, err := local.New(
		local.WithBin(bin),
		local.WithArgs(args),
	)
	if err != nil {
		log.Println("Cannot create local LLM:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	completion, err := llm.Call(context.Background(), prompt.Input)
	if err != nil {
		log.Println("Cannot get completion:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// remove the question if it appears in the response
	completion = strings.ReplaceAll(completion, prompt.Input, "")
	response := struct {
		Input    string `json:"input"`
		Response string `json:"response"`
	}{
		Input:    prompt.Input,
		Response: completion,
	}
	json.NewEncoder(w).Encode(response)
}

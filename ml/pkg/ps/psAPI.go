package ps

import (
	"encoding/json"
	"fmt"
	"github.com/diegostock12/thesis/ml/pkg/api"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)


func respondWithSuccess(w http.ResponseWriter, resp []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handleSchedulerResponse Handles the responses from the scheduler to the
// requests by the parameter servers to
func (ps *ParameterServer) handleSchedulerResponse(w http.ResponseWriter, r *http.Request)  {

	// Get the job that the new response is for
	vars := mux.Vars(r)
	jobId := vars["jobId"]

	// get the channel of the job
	ch, exists := ps.jobIndex[jobId]
	if !exists {
		ps.logger.Error("Received response for non-existing job",
			zap.String("id", jobId))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unpack the Response
	var resp api.ScheduleResponse
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ps.logger.Error("Could not read response body",
			zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		ps.logger.Error("Could not unmarshal the response json",
			zap.String("request", string(body)),
			zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}


	ps.logger.Debug("Received response from the scheduler, sending to job...",
		zap.Any("resp", resp))

	// Send the response to the channel
	ch <- &resp

	respondWithSuccess(w, []byte(jobId))
}

// handleScheduleRequest Handles the request of the scheduler to create a
// new training job. It creates a new parameter server thread and returns the id
// of the created parameeter server
// TODO the code for this is basically the code that is now present in the scheduler.go file
func (ps *ParameterServer) handleScheduleRequest(w http.ResponseWriter, r *http.Request)  {
	ps.logger.Debug("Processing request from the Scheduler")

	var task api.TrainTask
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ps.logger.Error("Could not read response body",
			zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &task)
	if err != nil {
		ps.logger.Error("Could not unmarshal the task json",
			zap.String("request", string(body)),
			zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ps.logger.Debug("Received task from the scheduler",
		zap.Any("task", task))

	// Create an ID for the new trainJob
	// and create the channel communicating
	// PS and the job
	jobId := uuid.New().String()[:8]
	ch := make(chan *api.ScheduleResponse)
	//jobId := "example"

	// TODO get a default parallelism
	// Create the train job and start serving
	job := newTrainJob(ps.logger, jobId, &task, ch)
	go job.serveTrainJob()
}

// Invoked by the serverless functions when they finish an epoch, should update the model
//func (ps *ParameterServer) handleFinish(w http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	funcId := vars["funcId"]
//
//	ps.logger.Info("Received finish signal from function, updating model",
//		zap.String("funcId", funcId))
//
//	w.WriteHeader(http.StatusOK)
//
//	// update the model with the new gradients
//	err := ps.model.Update(funcId)
//	if err != nil {
//		ps.logger.Error("Error while updating model",
//			zap.Error(err))
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//
//	// we have the json with the results in the body
//	var results map[string]float32
//	body, err := ioutil.ReadAll(r.Body)
//	if err != nil {
//		ps.logger.Error("Could not read the results from the training process",
//			zap.Error(err))
//		w.WriteHeader(http.StatusBadRequest)
//		return
//	}
//
//	err = json.Unmarshal(body, &results)
//	if err != nil {
//		ps.logger.Error("Could not unmarshal JSON",
//			zap.Error(err))
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//
//	ps.logger.Debug("Received results", zap.Any("res", results))
//
//	ps.logger.Info("Updated model weights")
//
//	// Change atomically the number of tasks to finish
//	// If there are no other tasks tell to the PS that it should
//	// Start the next epoch
//	ps.numLock.Lock()
//	defer ps.numLock.Unlock()
//	ps.toFinish -=1
//	if ps.toFinish == 0{
//		ps.epochChan <- struct{}{}
//	}
//
//}

// Returns the handler for calls from the functions
func (ps *ParameterServer) GetHandler() http.Handler {
	r := mux.NewRouter()
	//r.HandleFunc("/finish/{funcId}", ps.handleFinish).Methods("POST")
	r.HandleFunc("/start", ps.handleScheduleRequest).Methods("POST")
	r.HandleFunc("/update/{jobId}", ps.handleSchedulerResponse).Methods("POST")

	return r
}

// Start the API at the given port
// All of the parameter server threads share the same API, and
// they communicate through channels
func (ps *ParameterServer) Serve(port int) {
	ps.logger.Info("Starting Parameter Server api",
		zap.Int("port", port))

	addr := fmt.Sprintf(":%v", port)

	// Start serving the endpoint
	err := http.ListenAndServe(addr, ps.GetHandler())
	ps.logger.Fatal("Parameter Server API done",
		zap.Error(err))
}
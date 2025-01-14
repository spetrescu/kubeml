package api

import (
	corev1 "k8s.io/api/core/v1"
)

// Types used by the APIs of the controller and the scheduler

type (

	// TrainRequest is sent to the controller api to start a new training job
	// This is then embedded in the Train Task that is used by the PS
	TrainRequest struct {
		ModelType    string       `json:"model_type"`
		BatchSize    int          `json:"batch_size"`
		Epochs       int          `json:"epochs"`
		Dataset      string       `json:"dataset"`
		LearningRate float32      `json:"lr"`
		FunctionName string       `json:"function_name"`
		Options      TrainOptions `json:"options,omitempty"`
	}

	// TrainOptions allows users to define extra configurations for the
	// train job such as parallelism and validation options
	TrainOptions struct {
		DefaultParallelism int  `json:"default_parallelism"`
		StaticParallelism  bool `json:"static_parallelism"`
		ValidateEvery      int  `json:"validate_every"`
		// K is the parameter of the K-avg algorithm, after how many
		// updates we sync with the PS
		K int `json:"k"`
		// GoalAccuracy accuracy objective, after which we'll stop the training
		GoalAccuracy float64 `json:"goal_accuracy"`
	}

	// InferRequest is sent when wanting to get a result back from a trained network
	InferRequest struct {
		ModelId string        `json:"model_id"`
		Data    []interface{} `json:"data"`
	}

	// TrainTask associates the train request sent by the user
	// with the kubeml specific handler of the request or job
	// It is the main object exchanged by the Scheduler and parameter
	// server to schedule new parallelism
	TrainTask struct {
		Parameters TrainRequest `json:"request"`
		Job        JobInfo      `json:"job,omitempty"`
	}

	// JobInfo holds the information about the Job responsible
	// for training the network
	//
	// This includes training specific parameters such as the elapsed time,
	// parallelism and so on, but also lower level information such as this job's
	// pod and service definition definition
	// Also include the channel for backwards compatibility with the thread deploying
	// method and with a - so it is ignored
	JobInfo struct {
		JobId   string          `json:"id"`
		State   JobState        `json:"state"`
		Pod     *corev1.Pod     `json:"-"`
		Svc     *corev1.Service `json:"-"`
		Channel chan *JobState  `json:"-"`
	}

	// JobState holds the training specific variables of the job
	JobState struct {
		Parallelism int     `json:"parallelism"`
		ElapsedTime float64 `json:"elapsed_time"`
	}

	// JobHistory saves the intermediate results from the training process
	// epoch to epoch
	JobHistory struct {
		ValidationLoss []float64 `json:"validation_loss"`
		Accuracy       []float64 `json:"accuracy"`
		TrainLoss      []float64 `json:"train_loss"`
		Parallelism    []float64 `json:"parallelism"`
		EpochDuration  []float64 `json:"epoch_duration"`
	}

	// MetricUpdate is received by the parameter server from the train jobs
	// to refresh the metrics exposed to prometheus
	MetricUpdate struct {
		ValidationLoss float64 `json:"validations_loss"`
		Accuracy       float64 `json:"accuracy"`
		TrainLoss      float64 `json:"train_loss"`
		Parallelism    float64 `json:"parallelism"`
		EpochDuration  float64 `json:"epoch_duration"`
	}

	// A single datapoint plus label
	Datapoint struct {
		Features []float32 `json:"features"`
	}

	// History is the train and validation history of a
	// specific training job
	History struct {
		Id   string       `bson:"_id" json:"id"`
		Task TrainRequest `json:"task"`
		Data JobHistory   `json:"data,omitempty"`
	}

	// DatasetSummary describes the contents a kubeml dataset
	DatasetSummary struct {
		Name         string `json:"name"`
		TrainSetSize int64  `json:"train_set_size"`
		TestSetSize  int64  `json:"test_set_size"`
	}
)

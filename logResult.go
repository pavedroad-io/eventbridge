package main

import (
	"encoding/json"
	"fmt"
)

// Result for a given job
type logResult struct {
	job      []byte
	jobType  string
	metaData map[string]string
	payload  []byte
}

func (r *logResult) Job() []byte {
	return r.job
}

func (r *logResult) Decode() (Job, error) {
	if r.jobType == LogQueueJobType {
		jd := &logQueueJob{}
		err := json.Unmarshal(r.job, jd)
		if err != nil {
			return jd, nil
		}
		return nil, err
	} else if r.jobType == LogProcessorJobType {
		jd := &logProcessorJob{}
		err := json.Unmarshal(r.job, jd)
		if err != nil {
			return jd, nil
		}
		return nil, err
	}

	ne := fmt.Errorf("Unknown job type: %v", r.jobType)
	return nil, ne
}

// LogErrorResults generic handler for errors while
// running  a LogJob
func (r *logResult) LogErrorResults(job interface{}, callingerror error) (Result, error) {
	var jobD []byte

	// return errors in metadata
	md := make(map[string]string)

	md["original_error"] = callingerror.Error()

	jobD, err := json.Marshal(job)
	if err != nil {
		md["marshal_error"] = "Failed to marshal job data"

		// jobType is undefined if marshal fails
		md["jobtype_error"] = "No Job type error"
		r.jobType = "UNDEFINED"
	} else {
		r.jobType = job.(Job).Type()
	}
	r.metaData = md
	r.job = jobD

	return r, nil
}

func (r *logResult) MetaData() map[string]string {
	return r.metaData
}

func (r *logResult) Payload() []byte {
	return r.payload
}

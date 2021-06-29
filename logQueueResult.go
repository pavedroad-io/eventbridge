package main

// Result for a given job
type logQueueResult struct {
	job      *logQueueJob
	metaData map[string]string
	payload  []byte
}

func (r *logQueueResult) Job() Job {
	return r.job
}

func (r *logQueueResult) MetaData() map[string]string {
	return r.metaData
}

func (r *logQueueResult) Payload() []byte {
	return r.payload
}

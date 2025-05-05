package service

func Crawl(url string) {
	var depthCount uint8 = 0
	// Increment depth count when doc completed
	depthCount += 1
	// Iterate html on url extract links into ds
	// Iterate over ds and keep extracting

	// Also strip boilerplate and save in txt files
}

// Indexer service will then keep extracting txt from files and index
// This will talk to repo and update datastore with term document frequency matrix

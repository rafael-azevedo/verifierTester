# verifierTester
Quick testing execution looper that grabs information about the test runs into a json struct. 

## Installation
You must be logged into ocm stage 

```bash
make
```

This will build the binaries , get the cluster list from ocm , and tell you what to export
## Usage
 
To get the latest clusters from stage to test run 
```bash
make get-list
```
Export required variables
```bash
export CLUSTERLIST={PATH TO LAST CLUSTERLID}
export LEGACYBIN={PATH TO LEGACY BIN 0.34.x or earlier}
export LEGACYVERSION={LEGACY BINARY VERSION}
export PROBEBINARY={PATH TO NEW CURL BASED BIN 0.35.x or newer}
export PROBEVERSION={CURL BASED BINARY VERSION}
```
Execute the binary

This can take a while to run it executes the verifier 3x per cluster legacy x86, curl x86, curl arm64
```bash
bin/verifierTester.go -legacyBinary=$LEGACYBIN -legacyVersion=$LEGACYVERSION -probeBinary=$PROBEBINARY -probeVersion=$PROBEVERSION -clusterListFile=$CLUSTERLIST
```

It will output the results in a json struct to a file in the temp folder
```bash
"/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRun"
```

Json struct is the following 
```golang
// Define the struct to match the JSON structure
type VerifierRun struct {
	Duration      float64 `json:"duration"`
	CID           string  `json:"cid"`
	OsdctlVersion string  `json:"osdctl_version"`
	Probe         string  `json:"probe"`
	Arch          string  `json:"arch"`
	Output        string  `json:"output"`
	Error         bool    `json:"error"`
}
```
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

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

func main() {
	var legacyBinary, legacyVersion, probeBinary, probeVersion, clusterListFile, clusterID string
	flag.StringVar(&legacyBinary, "legacyBinary", "", "legacy binary path")
	flag.StringVar(&legacyVersion, "legacyVersion", "", "legacy binary version")
	flag.StringVar(&probeBinary, "probeBinary", "", "legacy binary path")
	flag.StringVar(&probeVersion, "probeVersion", "", "legacy binary version or commit")
	flag.StringVar(&clusterID, "clusterID", "", "clusterID")
	flag.StringVar(&clusterListFile, "clusterListFile", "", "clusterListFile")
	flag.Parse()

	fmt.Printf(" -----    Input Parameters    -----\n")
	fmt.Printf("------------------------------------\n")
	fmt.Printf(" -----    legacyBinary :: %s || legacyVersion   :: %s    -----\n", legacyBinary, legacyVersion)
	fmt.Printf(" -----    probeBinary :: %s || probeVersion   :: %s    -----\n", probeBinary, probeVersion)
	fmt.Printf(" -----    clusterListFile :: %s || clusterID   :: %s    -----\n", clusterListFile, clusterID)

	if legacyBinary == "" || probeBinary == "" || legacyVersion == "" || probeVersion == "" {
		fmt.Println("You must provide -legacyBinary, -legacyVersion, -probeBinary, -probeVersion and -clusterID for this program to function")
		os.Exit(1)
	}

	if (clusterID == "" && clusterListFile == "") || (clusterID != "" && clusterListFile != "") {
		fmt.Println("You must provide only one of the following clusterID or clusterList for this program to function")
		os.Exit(1)
	}
	runInputs := []struct {
		bin, version, prob, arch string
	}{
		{legacyBinary, legacyVersion, "legacy", "x86"},
		{probeBinary, probeVersion, "curl", "x86"},
		{probeBinary, probeVersion, "curl", "arm64"},
	}

	clusterIDs := []string{}
	if clusterID != "" {
		clusterIDs = append(clusterIDs, clusterID)
	}
	if clusterListFile != "" {
		content, err := os.ReadFile(clusterListFile)
		if err != nil {
			fmt.Printf("issue reading file, %s\n", err)
			os.Exit(1)
		}
		clusterIDs = append(clusterIDs, strings.Split(string(content), "\n")...)
	}

	outfile, err := os.Create("/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRun")
	if err != nil {
		fmt.Printf("issue writing file, %s\n", err)
		os.Exit(1)
	}

	outfileTmp, err := os.Create("/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRun_tmp")
	if err != nil {
		fmt.Printf("issue writing file, %s\n", err)
		os.Exit(1)
	}

	defer outfile.Close()
	defer outfileTmp.Close()
	// verifierRun Outputs holder
	verifierRuns := []VerifierRun{}
	for _, id := range clusterIDs {
		for _, inputs := range runInputs {
			fmt.Printf(" -----    Running Cluster - %s :: Binary- %s :: Arch %s    -----\n", id, inputs.bin, inputs.arch)
			v1, err := veriferToJson(inputs.bin, inputs.version, id, inputs.prob, inputs.arch)
			if err != nil {
				fmt.Printf(" -----    Issue running verifier :: %s    -----\n", err)
			}
			fmt.Printf(" -----    Run Done    -----\n\n")
			verifierRuns = append(verifierRuns, v1)

			// write to tmp file
			data, err := json.MarshalIndent(v1, "", " ")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Fprintf(outfileTmp, "\n %s ,", string(data))
		}

	}

	data, err := json.MarshalIndent(verifierRuns, "", " ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	written, err := outfile.WriteString(string(data))
	if err != nil {
		fmt.Printf("issue writing file, %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("%d bytes written to file %s", written, outfile.Name())
}

func veriferToJson(binary, osdctlVersion, clusterID, probe, arch string) (VerifierRun, error) {
	start := time.Now()
	buf := new(bytes.Buffer)
	verifierRun := VerifierRun{
		CID:           clusterID,
		OsdctlVersion: osdctlVersion,
		Probe:         probe,
		Arch:          arch,
		Error:         false,
	}
	err := execVerifier(binary, clusterID, arch, buf)
	if err != nil {
		verifierRun.Duration = time.Since(start).Seconds()
		verifierRun.Output = buf.String()
		verifierRun.Error = true
		return verifierRun, err
	}

	verifierRun.Duration = time.Since(start).Seconds()
	verifierRun.Output = buf.String()
	return verifierRun, err
}

func execVerifier(binary, clusterID, cpuArch string, buf *bytes.Buffer) error {
	//Setup Commands
	sendN := exec.Command("echo", "N")
	//fmt.Printf("---- Executing Command ----\n %s %s %s %s %s %s %s %s %s\n -------------\n", binary, "network", "verify-egress", "--cluster-id", clusterID, "--egress-timeout", "5s", "-S", "--debug")
	fmt.Printf("Running on clusterID %s \n", clusterID)
	verifierCMD := exec.Command(binary, "network", "verify-egress", "--cluster-id", clusterID, "--egress-timeout", "5s", "-S")
	if cpuArch == "amd64" {
		verifierCMD = exec.Command(binary, "network", "verify-egress", "--cluster-id", clusterID, "--cpu-arch", cpuArch, "--egress-timeout", "5s", "-S")

	}
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()
	sendN.Stdout = w
	err = sendN.Start()
	if err != nil {
		return err
	}
	defer sendN.Wait()
	w.Close()

	verifierCMD.Stdin = r

	// output in line
	oBuf := io.MultiWriter(os.Stdout, buf)
	verifierCMD.Stdout = oBuf
	verifierCMD.Stderr = oBuf
	err = verifierCMD.Run()
	if err != nil {
		return err
	}
	return nil
}

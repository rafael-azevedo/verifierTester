package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
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
type stringFlag struct {
	set   bool
	value string
}

func (sf *stringFlag) Set(x string) error {
	sf.value = x
	sf.set = true
	return nil
}

func (sf *stringFlag) String() string {
	return sf.value
}

var legacyBinary, legacyVersion, probeBinary, probeVersion, clusterListFile, clusterID, convert, jsonFile stringFlag

func init() {
	flag.Var(&convert, "convert", "set this flag to convert jsonFile to csv")
	flag.Var(&legacyBinary, "legacyBinary", "legacy binary path")
	flag.Var(&legacyVersion, "legacyVersion", "legacy binary version")
	flag.Var(&probeBinary, "probeBinary", "legacy binary path")
	flag.Var(&probeVersion, "probeVersion", "legacy binary version or commit")
	flag.Var(&clusterID, "clusterID", "clusterID")
	flag.Var(&clusterListFile, "clusterListFile", "path to list of cluster ids")
	flag.Var(&jsonFile, "jsonFile", "path to json file to convert to csv")
}
func main() {
	flag.Parse()
	switch {
	case convert.set:
		if !jsonFile.set {
			fmt.Println("no -jsonFile provided")
			os.Exit(1)
		}
		err := convertFiletoCSV(jsonFile.value)
		if err != nil {
			fmt.Printf("error converting to csv :: %s \n", err)
		}
	default:
		RunTests()
	}
}

func RunTests() error {
	fmt.Printf(" -----    Input Parameters    -----\n")
	fmt.Printf("------------------------------------\n")
	fmt.Printf(" -----    legacyBinary :: %s || legacyVersion   :: %s    -----\n", legacyBinary.value, legacyVersion.value)
	fmt.Printf(" -----    probeBinary :: %s || probeVersion   :: %s    -----\n", probeBinary.value, probeVersion.value)
	fmt.Printf(" -----    clusterListFile :: %s || clusterID   :: %s    -----\n", clusterListFile.value, clusterID.value)

	if !legacyBinary.set || !probeBinary.set || !legacyVersion.set || !probeVersion.set {
		return errors.New("you must provide -legacyBinary, -legacyVersion, -probeBinary, -probeVersion and -clusterID for this program to function")
	}

	if (clusterID.set && clusterListFile.set) || (!clusterID.set && !clusterListFile.set) {
		return errors.New("you must provide only one of the following clusterID or clusterList for this program to function")
	}

	runInputs := []struct {
		bin, version, prob, arch string
	}{
		{legacyBinary.value, legacyVersion.value, "legacy", "x86"},
		{probeBinary.value, probeVersion.value, "curl", "x86"},
		{probeBinary.value, probeVersion.value, "curl", "arm64"},
	}

	clusterIDs := []string{}
	if clusterID.set {
		clusterIDs = append(clusterIDs, clusterID.value)
	}
	if clusterListFile.set {
		content, err := os.ReadFile(clusterListFile.value)
		if err != nil {
			return fmt.Errorf("issue reading file, %s", err)
		}
		clusterIDs = append(clusterIDs, strings.Split(string(content), "\n")...)
	}

	outfile, err := os.Create("/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRun")
	if err != nil {
		return fmt.Errorf("issue writing file, %s", err)
	}

	outfileTmp, err := os.Create("/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRun_tmp")
	if err != nil {
		return fmt.Errorf("issue writing file, %s", err)
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
				return err
			}
			fmt.Fprintf(outfileTmp, "\n %s ,", string(data))
		}

	}

	data, err := json.MarshalIndent(verifierRuns, "", " ")
	if err != nil {
		return err
	}

	written, err := outfile.WriteString(string(data))
	if err != nil {
		return fmt.Errorf("issue writing file, %s", err)
	}
	fmt.Printf("%d bytes written to file %s", written, outfile.Name())
	return nil
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

func (VerifierRun) CSVheader(w io.Writer) {
	cw := csv.NewWriter(w)
	cw.Write([]string{"duration", "cid", "osdctl_version", "probe", "arch", "output", "error"})
	cw.Flush()
}

func (vr VerifierRun) CSVrow(w io.Writer) {
	cw := csv.NewWriter(w)
	cw.Write([]string{strconv.FormatFloat(vr.Duration, 'f', -1, 64), vr.CID, vr.OsdctlVersion, vr.Probe, vr.Arch, vr.Output, strconv.FormatBool(vr.Error)})
	cw.Flush()
}

func verifierRunsToCSV(verifierRuns []VerifierRun, w io.Writer) {
	verifierRuns[0].CSVheader(w)
	for _, vr := range verifierRuns {
		vr.CSVrow(w)
	}
}

func convertFiletoCSV(fileName string) error {
	// vsFile := struct {
	// 	VS []VerifierRun
	// }{}
	var vsFile []VerifierRun
	content, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &vsFile)
	if err != nil {
		return err
	}

	csvFile, err := os.Create("/tmp/" + time.Now().Format(time.RFC3339) + "_verifierRuns.csv")
	if err != nil {
		return err
	}
	defer csvFile.Close()

	verifierRunsToCSV(vsFile, csvFile)
	fmt.Printf("------------------------------------\n")
	fmt.Printf("Converted %s to csv %s\n", fileName, csvFile.Name())
	fmt.Printf("------------------------------------\n")
	return nil
}

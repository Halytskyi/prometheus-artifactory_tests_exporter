/*
* Copyright (C) 2019 Oleh Halytskyi
*
* This software may be modified and distributed under the terms
* of the Apache license. See the LICENSE file for details.
*
 */

package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
)

var logLevel string = "none"
var version string
var buildDate string
var testFilesDir string
var pushDuration map[string]float64
var pushSuccess map[string]float64
var pullDuration map[string]float64
var pullSuccess map[string]float64

func hash_file_md5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

func makeTests(artifactoryServer ArtifactoryParams, testsInterval float64, testFiles map[string]TestFileParams, filesChecksum map[string]string, filesTimeoutPush map[string]float64, filesTimeoutPull map[string]float64) {
	artifactoryPath := artifactoryServer.URL + "/" + artifactoryServer.RepoPath
	pushDuration = make(map[string]float64)
	pushSuccess = make(map[string]float64)
	pullDuration = make(map[string]float64)
	pullSuccess = make(map[string]float64)

	for {
		if logLevel == "info" || logLevel == "debug" {
			glog.Infoln("---====== Start tests ======---")
		}
		start_tests := time.Now()
		for fileName, fileParams := range testFiles {
			artifactoryFullPath := artifactoryPath + "/" + fileName
			fileFullPath := testFilesDir + "/" + fileName

			// Push test file
			testFile, err := os.Open(fileFullPath)
			if err == nil {
				ctx_push, _ := context.WithTimeout(context.Background(), time.Duration(filesTimeoutPush[fileName]*float64(time.Second)))
				if logLevel == "info" || logLevel == "debug" {
					glog.Infoln("Start push file '" + fileName + "'")
				}
				start_push := time.Now()
				req_push, err := http.NewRequest("PUT", artifactoryFullPath, testFile)
				if err == nil {
					resp_push, err := http.DefaultClient.Do(req_push.WithContext(ctx_push))
					if err == nil {
						if resp_push.StatusCode == 201 {
							pushDuration[fileName] = time.Since(start_push).Seconds()
							pushSuccess[fileName] = 1
							if logLevel == "info" || logLevel == "debug" {
								glog.Infoln("'"+fileName+"' push duration:", pushDuration[fileName])
							}
						} else {
							pushDuration[fileName] = 0
							pushSuccess[fileName] = 0
							if logLevel == "error" || logLevel == "debug" {
								glog.Errorln("Response code:", resp_push.StatusCode)
							}
						}
						resp_push.Body.Close()
					} else {
						pushDuration[fileName] = 0
						pushSuccess[fileName] = 0
						if logLevel == "error" || logLevel == "debug" {
							glog.Errorln(err)
						}
					}
				} else {
					pushDuration[fileName] = 0
					pushSuccess[fileName] = 0
					if logLevel == "error" || logLevel == "debug" {
						glog.Errorln(err)
					}
				}
			} else {
				pushDuration[fileName] = 0
				pushSuccess[fileName] = 0
				if logLevel == "error" || logLevel == "debug" {
					glog.Errorln(err)
				}
			}
			testFile.Close()

			// Pull test file
			ctx_pull, _ := context.WithTimeout(context.Background(), time.Duration(filesTimeoutPull[fileName]*float64(time.Second)))
			if logLevel == "info" || logLevel == "debug" {
				glog.Infoln("Start pull file '" + fileName + "'")
			}
			start_pull := time.Now()
			req_pull, err := http.NewRequest("GET", artifactoryFullPath, nil)
			if err == nil {
				resp_pull, err := http.DefaultClient.Do(req_pull.WithContext(ctx_pull))
				if err == nil {
					if resp_pull.StatusCode == 200 {
						// Create the file
						fileDownloaded, err := os.Create(fileFullPath + "-downloaded")
						if err == nil {
							_, err = io.Copy(fileDownloaded, resp_pull.Body)
							if err == nil {
								// Verify downloaded file checksum
								if fileParams.VerifyChecksum {
									hash, err := hash_file_md5(fileFullPath + "-downloaded")
									if err == nil {
										if logLevel == "info" || logLevel == "debug" {
											glog.Infoln("'"+fileName+"' hash: ", filesChecksum[fileName])
											glog.Infoln("'"+fileName+"-downloaded' hash: ", hash)
										}
										if hash == filesChecksum[fileName] {
											pullDuration[fileName] = time.Since(start_pull).Seconds()
											pullSuccess[fileName] = 1
											if logLevel == "info" || logLevel == "debug" {
												glog.Infoln("'"+fileName+"' pull duration:", pullDuration[fileName])
											}
										} else {
											pullDuration[fileName] = 0
											pullSuccess[fileName] = 0
											if logLevel == "error" || logLevel == "debug" {
												glog.Errorln("Checksum for file '" + fileName + "-downloaded' not the same as for original")
											}
										}
									} else {
										pullDuration[fileName] = 0
										pullSuccess[fileName] = 0
										if logLevel == "error" || logLevel == "debug" {
											glog.Errorln(err)
										}
									}
								} else {
									pullDuration[fileName] = time.Since(start_pull).Seconds()
									pullSuccess[fileName] = 1
									if logLevel == "info" || logLevel == "debug" {
										glog.Infoln("'"+fileName+"' pull duration:", pullDuration[fileName])
									}
								}
							} else {
								pullDuration[fileName] = 0
								pullSuccess[fileName] = 0
								if logLevel == "error" || logLevel == "debug" {
									glog.Errorln(err)
								}
							}
						} else {
							pullDuration[fileName] = 0
							pullSuccess[fileName] = 0
							if logLevel == "error" || logLevel == "debug" {
								glog.Errorln(err)
							}
						}
						fileDownloaded.Close()
					} else {
						pullDuration[fileName] = 0
						pullSuccess[fileName] = 0
						if logLevel == "error" || logLevel == "debug" {
							glog.Errorln("Response code:", resp_pull.StatusCode)
						}
					}
					resp_pull.Body.Close()
				} else {
					pullDuration[fileName] = 0
					pullSuccess[fileName] = 0
					if logLevel == "error" || logLevel == "debug" {
						glog.Errorln(err)
					}
				}
			} else {
				pullDuration[fileName] = 0
				pullSuccess[fileName] = 0
				if logLevel == "error" || logLevel == "debug" {
					glog.Errorln(err)
				}
			}
		}
		timeToWait := time.Duration(testsInterval*float64(time.Second)) - time.Since(start_tests)
		if logLevel == "info" || logLevel == "debug" {
			glog.Infoln("---====== Finish tests ======---")
			glog.Infoln("Waiting", timeToWait)
		}
		time.Sleep(timeToWait)
	}
}

func ArtifactoryCollector(testFiles map[string]TestFileParams, registry *prometheus.Registry, artifactoryPushDurationHistograms map[string]*prometheus.HistogramVec, artifactoryPullDurationHistograms map[string]*prometheus.HistogramVec) {
	for fileName, fileParams := range testFiles {
		pushDurationGaugeName := "artifactory_test_push_" + strconv.Itoa(fileParams.Size) + "mb_duration_seconds"
		artifactoryPushDurationGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: pushDurationGaugeName,
			Help: "How long the Artifactory push-test took to complete in seconds",
		}, []string{"file_name"})

		pullDurationGaugeName := "artifactory_test_pull_" + strconv.Itoa(fileParams.Size) + "mb_duration_seconds"
		artifactoryPullDurationGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: pullDurationGaugeName,
			Help: "How long the Artifactory pull-test took to complete in seconds",
		}, []string{"file_name"})

		successPushGaugeName := "artifactory_test_push_" + strconv.Itoa(fileParams.Size) + "mb_success"
		successPushGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: successPushGaugeName,
			Help: "Displays whether or not the Artifactory push-test was a success",
		}, []string{"file_name"})

		successPullGaugeName := "artifactory_test_pull_" + strconv.Itoa(fileParams.Size) + "mb_success"
		successPullGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: successPullGaugeName,
			Help: "Displays whether or not the Artifactory pull-test was a success",
		}, []string{"file_name"})

		registry.MustRegister(artifactoryPushDurationHistograms[fileName])
		registry.MustRegister(artifactoryPullDurationHistograms[fileName])
		registry.MustRegister(artifactoryPushDurationGauge)
		registry.MustRegister(artifactoryPullDurationGauge)
		registry.MustRegister(successPushGauge)
		registry.MustRegister(successPullGauge)

		artifactoryPushDurationHistograms[fileName].WithLabelValues(fileName).Observe(pushDuration[fileName])
		artifactoryPullDurationHistograms[fileName].WithLabelValues(fileName).Observe(pullDuration[fileName])
		artifactoryPushDurationGauge.WithLabelValues(fileName).Set(pushDuration[fileName])
		artifactoryPullDurationGauge.WithLabelValues(fileName).Set(pullDuration[fileName])
		successPushGauge.WithLabelValues(fileName).Set(pushSuccess[fileName])
		successPullGauge.WithLabelValues(fileName).Set(pullSuccess[fileName])
	}
}

func artifactoryTestHandler(w http.ResponseWriter, r *http.Request, timeoutHandlerSeconds float64, testFiles map[string]TestFileParams, artifactoryPushDurationHistograms map[string]*prometheus.HistogramVec, artifactoryPullDurationHistograms map[string]*prometheus.HistogramVec) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutHandlerSeconds*float64(time.Second)))
	defer cancel()
	r = r.WithContext(ctx)

	registry := prometheus.NewRegistry()
	ArtifactoryCollector(testFiles, registry, artifactoryPushDurationHistograms, artifactoryPullDurationHistograms)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func createTestFiles(fileName string, fileFullPath string, fileSize int) error {
	createFile := true

	type payload struct {
		One   float32
		Two   float64
		Three uint32
	}

	// Verify if defined correct file size
	if fileSize <= 0 {
		return errors.New("File size can't be 0 or less")
	}

	// Create dir for test files if it not exist
	if _, err := os.Stat(testFilesDir); os.IsNotExist(err) {
		err = os.Mkdir(testFilesDir, 0755)
		if err != nil {
			return err
		}
	}

	// Verify file size if it exists
	if file_stat, err := os.Stat(fileFullPath); err == nil {
		fsize := int(file_stat.Size() / 1024 / 1024)
		if fsize == fileSize {
			createFile = false
		}
	}

	if createFile {
		file, err := os.Create(fileFullPath)
		defer file.Close()
		if err != nil {
			return err
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < fileSize*128*512; i++ {
			s := &payload{
				r.Float32(),
				r.Float64(),
				r.Uint32(),
			}
			var bin_buf bytes.Buffer
			binary.Write(&bin_buf, binary.BigEndian, s)
			_, err := file.Write(bin_buf.Bytes())
			if err != nil {
				return err
			}
		}
		glog.Infoln("Created file:", fileFullPath)
	}
	return nil
}

func main() {
	var (
		sc            = &SafeConfig{C: &Config{}}
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9702").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		configFile    = kingpin.Flag("config.file", "Artifactory Test Exporter configuration file.").Default("artifactory-tests.yml").String()
	)

	flag.Set("logtostderr", "true")
	flag.Parse()

	kingpin.Version(version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	glog.Infoln("Starting artifactory-tests_exporter, version", version)
	glog.Infoln("Build date:", buildDate)

	if err := sc.LoadConfig(*configFile); err != nil {
		glog.Fatal("Error loading config", err)
	}
	sc.Lock()
	conf := sc.C
	sc.Unlock()
	glog.Infoln("Loaded config file")

	// Tests interval
	testsInterval := 60.0
	if conf.Interval.Seconds() != 0 {
		if conf.Interval.Seconds() >= testsInterval {
			testsInterval = conf.Interval.Seconds()
		} else {
			glog.Fatal("Tests interval should be more or equal 60 seconds")
		}
	} else {
		glog.Infoln("Tests interval:", testsInterval, "seconds")
	}

	// Timeout for handler
	timeoutHandlerSeconds := 5.0
	if conf.Timeout.Seconds() != 0 {
		if conf.Timeout.Seconds() <= timeoutHandlerSeconds && conf.Timeout.Seconds() > 0 {
			timeoutHandlerSeconds = conf.Timeout.Seconds()
		} else {
			glog.Fatal("Handler timeout should be less or equal 5 seconds")
		}
	} else {
		glog.Infoln("Handler timeout:", timeoutHandlerSeconds, "seconds")
	}

	// Debug
	logLevel = conf.LogLevel

	// Artifactory server parameters
	artifactoryParams := conf.Artifactory

	// Path for test files
	testFilesDir = conf.TestFilesPath
	if testFilesDir == "" {
		testFilesDir = "/opt/prometheus/prometheus-artifactory-tests-exporter/test-files"
	}

	// Parse test files parameters
	artifactoryPushDurationHistograms := make(map[string]*prometheus.HistogramVec)
	artifactoryPullDurationHistograms := make(map[string]*prometheus.HistogramVec)
	filesTimeoutPush := make(map[string]float64)
	filesTimeoutPull := make(map[string]float64)
	filesChecksum := make(map[string]string)
	testFiles := conf.TestFiles
	// Maximum file timeout for full-cycle interval should be: testsInterval / 2 (push and pull tests) / num files - 1 second (for tests running)
	maxFileTimeout := testsInterval/2.0/float64(len(testFiles)) - 1
	for fileName, fileParams := range testFiles {
		fileFullPath := testFilesDir + "/" + fileName

		// Create test files
		err := createTestFiles(fileName, fileFullPath, fileParams.Size)
		if err != nil {
			glog.Fatal(err)
		}

		// Histograms for Push tests
		pushDurationHistogramName := "artifactory_test_push_" + strconv.Itoa(fileParams.Size) + "mb_duration_seconds_histogram"
		artifactoryPushDurationHistograms[fileName] = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    pushDurationHistogramName,
			Help:    "Histogram for the Artifactory push-test",
			Buckets: fileParams.HistogramBucketPush,
		}, []string{"file_name"})

		// Histograms for Pull tests
		pullDurationHistogramName := "artifactory_test_pull_" + strconv.Itoa(fileParams.Size) + "mb_duration_seconds_histogram"
		artifactoryPullDurationHistograms[fileName] = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    pullDurationHistogramName,
			Help:    "Histogram for the Artifactory pull-test",
			Buckets: fileParams.HistogramBucketPull,
		}, []string{"file_name"})

		// Push and pull timeouts
		if fileParams.TimeoutPush.Seconds() != 0 {
			if fileParams.TimeoutPush.Seconds() <= maxFileTimeout && fileParams.TimeoutPush.Seconds() > 0 {
				filesTimeoutPush[fileName] = fileParams.TimeoutPush.Seconds()
			} else {
				glog.Fatal("For full-cycle interval 'timeout_push' parameter should be less or equal '" + strconv.FormatFloat(maxFileTimeout, 'f', 0, 64) + "' and more than '0'")
			}
		} else {
			filesTimeoutPush[fileName] = maxFileTimeout
			glog.Infoln("Push timeout for '"+fileName+"':", maxFileTimeout, "seconds")
		}
		if fileParams.TimeoutPull.Seconds() != 0 {
			if fileParams.TimeoutPull.Seconds() <= maxFileTimeout && fileParams.TimeoutPull.Seconds() > 0 {
				filesTimeoutPull[fileName] = fileParams.TimeoutPull.Seconds()
			} else {
				glog.Fatal("For full-cycle interval 'timeout_pull' parameter should be less or equal '" + strconv.FormatFloat(maxFileTimeout, 'f', 0, 64) + "' and more than '0'")
			}
		} else {
			filesTimeoutPull[fileName] = maxFileTimeout
			glog.Infoln("Pull timeout for '"+fileName+"':", maxFileTimeout, "seconds")
		}

		// File checksum
		if fileParams.VerifyChecksum {
			hash, err := hash_file_md5(fileFullPath)
			if err == nil {
				filesChecksum[fileName] = hash
			} else {
				glog.Fatal(err)
			}
		}
	}

	// Custom handler
	_metricsPath := conf.MetricsPath
	if _metricsPath == "" {
		_metricsPath = *metricsPath
	}
	glog.Infoln("Metrics path", _metricsPath)
	http.HandleFunc(_metricsPath, func(w http.ResponseWriter, r *http.Request) {
		artifactoryTestHandler(w, r, timeoutHandlerSeconds, testFiles, artifactoryPushDurationHistograms, artifactoryPullDurationHistograms)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Artifactory Test Exporter</title></head>
			<body>
			<h1>Artifactory Test Exporter</h1>
			<p><a href="` + _metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	// Run tests
	go makeTests(artifactoryParams, testsInterval, testFiles, filesChecksum, filesTimeoutPush, filesTimeoutPull)

	_listenAddress := conf.ListenAddress
	if _listenAddress == "" {
		_listenAddress = *listenAddress
	}
	glog.Infoln("Listening on", _listenAddress)
	if err := http.ListenAndServe(_listenAddress, nil); err != nil {
		glog.Fatal(err)
	}
}

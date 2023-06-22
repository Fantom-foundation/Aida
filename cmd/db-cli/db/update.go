package db

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/klauspost/compress/gzip"
	"github.com/urfave/cli/v2"
)

const maxNumberOfDownloadAttempts = 5

// UpdateCommand downloads aida-db and new patches
var UpdateCommand = cli.Command{
	Action: update,
	Name:   "update",
	Usage:  "download aida-db patches",
	Flags: []cli.Flag{
		&utils.AidaDbFlag,
		&utils.ChainIDFlag,
		&logger.LogLevelFlag,
		&utils.CompactDbFlag,
		&utils.DbTmpFlag,
	},
	Description: ` 
Updates aida-db by downloading patches from aida-db generation server.
`,
}

// update updates aida-db by downloading patches from aida-db generation server.
func update(ctx *cli.Context) error {
	cfg, err := utils.NewConfig(ctx, utils.NoArgs)
	if err != nil {
		return err
	}
	if err = Update(cfg); err != nil {
		return err
	}

	return printMetadata(cfg.AidaDb)
}

// Update implements updating command to be called from various commands and automatically downloads aida-db patches.
func Update(cfg *utils.Config) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Update")

	targetMD, err := getTargetDbBlockRange(cfg)
	if err != nil {
		return fmt.Errorf("unable retrieve aida-db metadata; %v", err)
	}

	log.Noticef("lastAidaDbBlock %v", targetMD.lastBlock)

	// retrieve available patches from aida-db generation server
	patches, err := retrievePatchesToDownload(targetMD.lastBlock)
	if err != nil {
		return fmt.Errorf("unable to prepare list of aida-db patches for download; %v", err)
	}

	if len(patches) == 0 {
		log.Warning("No new patches to download are available")
		MustCloseDB(targetMD.db)
		return nil
	}

	// create a parents of temporary directory
	err = os.MkdirAll(cfg.DbTmp, 0755)
	if err != nil {
		return fmt.Errorf("failed to create %s directory; %s", cfg.DbTmp, err)
	}

	log.Noticef("Downloading Aida-db - %d new patches", len(patches))

	// we need to know whether Db is new for metadata
	err = patchesDownloader(cfg, patches, targetMD, targetMD.lastBlock == 0)
	if err != nil {
		return err
	}

	log.Notice("Aida-db update finished successfully")

	return nil
}

// getTargetDbBlockRange initialize aidaMetadata of targetDB
func getTargetDbBlockRange(cfg *utils.Config) (*aidaMetadata, error) {
	var (
		firstAidaDbBlock, lastAidaDbBlock uint64
	)

	// load stats of current aida-db to download just latest patches
	_, err := os.Stat(cfg.AidaDb)
	if err != nil {
		if os.IsNotExist(err) {
			// aida-db does not exist, download all available patches
		} else {
			return nil, err
		}
	} else {
		// load last block from existing aida-db metadata
		firstAidaDbBlock, lastAidaDbBlock, err = findBlockRangeInSubstate(cfg.AidaDb)
		if err != nil {
			return nil, fmt.Errorf("using corrupted aida-db database; %v", err)
		}
	}

	// aida-db already exists appending only new patches
	// open targetDB
	targetDb, err := rawdb.NewLevelDBDatabase(cfg.AidaDb, 1024, 100, "profiling", false)
	if err != nil {
		return nil, fmt.Errorf("can't open aidaDb; %v", err)
	}

	targetMD := newAidaMetadata(targetDb, cfg.LogLevel)
	if err = targetMD.setBlockRange(firstAidaDbBlock, lastAidaDbBlock); err != nil {
		return nil, err
	}

	return targetMD, nil
}

// patchesDownloader processes patch names to download then download them in pipelined process
func patchesDownloader(cfg *utils.Config, patches []string, targetMD *aidaMetadata, isNewDb bool) error {
	// create channel to push patch labels trough channel
	patchesChan := pushStringsToChannel(patches)

	// download patches
	downloadedPatchChan, errChan := downloadPatch(cfg, patchesChan)

	// decompress downloaded patches
	decompressedPatchChan, errDecompressChan := decompressPatch(cfg, downloadedPatchChan, errChan)

	// merge decompressed patches
	err := mergePatch(cfg, decompressedPatchChan, errDecompressChan, targetMD, isNewDb)
	if err != nil {
		return err
	}

	if err = targetMD.db.Close(); err != nil {
		targetMD.log.Warningf("patchesDownloader: cannot close targetDb; %v", err)
	}

	return nil
}

// mergePatch takes decompressed patches and merges them into aida-db
func mergePatch(cfg *utils.Config, decompressChan chan string, errChan chan error, targetMD *aidaMetadata, isNewDb bool) error {
	for {
		select {
		case err, ok := <-errChan:
			{
				if ok {
					return err
				}
			}
		case extractedPatchPath, ok := <-decompressChan:
			{
				if !ok {
					return nil
				}
				// merge newly extracted patch

				targetDb := targetMD.db

				patchDb, err := rawdb.NewLevelDBDatabase(extractedPatchPath, 1024, 100, "profiling", false)
				if err != nil {
					return fmt.Errorf("cannot open targetDb; %v", err)
				}

				patchMD := newAidaMetadata(patchDb, cfg.LogLevel)

				firstBlock, firstEpoch, err := targetMD.checkUpdateMetadata(isNewDb, cfg, patchMD)
				if err != nil {
					return err
				}

				// after inserting first patch, db is no longer new
				isNewDb = false

				m := newMerger(cfg, targetDb, []ethdb.Database{patchDb}, []string{extractedPatchPath}, nil)

				err = m.merge()
				if err != nil {
					return fmt.Errorf("unable to merge %v; %v", extractedPatchPath, err)
				}

				err = targetMD.setAllMetadata(firstBlock, patchMD.lastBlock, firstEpoch, patchMD.lastEpoch, patchMD.chainId, genType)
				if err != nil {
					return err
				}

				m.closeSourceDbs()

				// remove patch
				err = os.RemoveAll(extractedPatchPath)
				if err != nil {
					return err
				}
			}
		}
	}
}

// decompressPatch takes tar.gz archives and decompresses them, then sends them for further processing
func decompressPatch(cfg *utils.Config, patchChan chan string, errChan chan error) (chan string, chan error) {
	log := logger.NewLogger(cfg.LogLevel, "Decompress patch")
	decompressedPatchChan := make(chan string, 1)
	errDecompressChan := make(chan error, 1)

	go func() {
		defer close(decompressedPatchChan)
		defer close(errDecompressChan)
		for {
			select {
			case err, ok := <-errChan:
				{
					if ok {
						errDecompressChan <- err
						return
					}
				}
			case fileName, ok := <-patchChan:
				{
					if !ok {
						return
					}
					log.Debugf("Decompressing %v...", fileName)

					compressedPatchPath := filepath.Join(cfg.DbTmp, fileName)
					err := extractTarGz(compressedPatchPath, cfg.DbTmp)
					if err != nil {
						errDecompressChan <- err
						return
					}

					// extracted patch is folder without the .tar.gz extension
					extractedPatchPath := strings.TrimSuffix(compressedPatchPath, ".tar.gz")

					decompressedPatchChan <- extractedPatchPath
					// remove compressed patch
					err = os.RemoveAll(compressedPatchPath)
					if err != nil {
						errDecompressChan <- err
						return
					}
				}

			}
		}
	}()

	return decompressedPatchChan, errDecompressChan

}

// downloadPatch downloads patches from server and sends them towards decompressor
func downloadPatch(cfg *utils.Config, patchesChan chan string) (chan string, chan error) {
	log := logger.NewLogger(cfg.LogLevel, "Download patch")
	downloadedPatchChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(downloadedPatchChan)
		defer close(errChan)
		for {
			fileName, ok := <-patchesChan
			if !ok {
				return
			}
			log.Debugf("Downloading %s...", fileName)
			patchUrl := utils.AidaDbRepositoryUrl + "/" + fileName
			compressedPatchPath := filepath.Join(cfg.DbTmp, fileName)
			err := downloadFile(compressedPatchPath, cfg.DbTmp, patchUrl)
			if err != nil {
				errChan <- fmt.Errorf("unable to download %s; %v", patchUrl, err)
			}
			log.Debugf("Downloaded %s", fileName)

			patchMd5Url := patchUrl + ".md5"

			// WARNING don't rewrite the following md5 check into separate thread,
			// because having two patches at same time might be too big for somebodies disk space
			md5Expected, err := loadExpectedMd5(patchMd5Url)
			if err != nil {
				errChan <- err
				return
			}

			log.Debugf("Calculating %s md5", fileName)
			md5, err := calculateMD5Sum(compressedPatchPath)
			if err != nil {
				errChan <- fmt.Errorf("archive %v; unable to calculate md5sum; %v", fileName, err)
				return
			}

			// Compare whether downloaded file matches expected md5
			if strings.Compare(md5, md5Expected) != 0 {
				errChan <- fmt.Errorf("archive %v doesn't have matching md5; archive %v, expected %v", fileName, md5, md5Expected)
				return
			}

			downloadedPatchChan <- fileName
		}
	}()
	return downloadedPatchChan, errChan
}

// loadExpectedMd5 loads md5 of file at given url
func loadExpectedMd5(patchMd5Url string) (string, error) {
	var buf bytes.Buffer

	writer := bufio.NewWriter(&buf)

	err := getFileContentsFromUrl(patchMd5Url, 0, writer)
	if err != nil {
		return "", fmt.Errorf("unable to download %s; %v", patchMd5Url, err)
	}

	// Flush the buffered writer to ensure all data is written to the buffer
	err = writer.Flush()
	if err != nil {
		return "", fmt.Errorf("flushing writer; %v", err)
	}

	// Get the written content as a string
	md5Expected := buf.String()

	// trimming whitespaces
	return strings.TrimSpace(md5Expected), nil
}

// pushStringsToChannel used to pipe strings into channel
func pushStringsToChannel(strings []string) chan string {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		for _, str := range strings {
			ch <- str
		}
	}()
	return ch
}

// retrievePatchesToDownload retrieves all available patches from aida-db generation server.
func retrievePatchesToDownload(startDownloadFromBlock uint64) ([]string, error) {
	// download list of available patches
	patches, err := downloadPatchesJson()
	if err != nil {
		return nil, fmt.Errorf("unable to download patches: %v", err)
	}

	// list of patches to be downloaded
	var fileNames = make([]string, 0)

	for _, patch := range patches {
		patchMap, ok := patch.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid patch in json; %v", patch)
		}

		// retrieve toBlock end of patch
		patchToBlockStr, ok := patchMap["toBlock"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid fromEpoch attributes in patch; %v", patchMap)
		}
		patchToBlock, err := strconv.ParseUint(patchToBlockStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse uint64 fromEpoch attribute in patch %v; %v", patchMap, err)
		}
		if patchToBlock <= startDownloadFromBlock {
			// skip every patch which is sooner than previous last block
			continue
		}

		fileName, ok := patchMap["fileName"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid fileName attributes in patch; %v", patchMap)
		}
		fileNames = append(fileNames, fileName)
	}

	sort.Strings(fileNames)

	return fileNames, nil
}

// downloadPatchesJson downloads list of available patches from aida-db generation server.
func downloadPatchesJson() ([]interface{}, error) {
	// Make the HTTP GET request
	patchesUrl := utils.AidaDbRepositoryUrl + "/patches.json"
	response, err := http.Get(patchesUrl)
	if err != nil {
		return nil, fmt.Errorf("error making GET request for %s: %v", patchesUrl, err)
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Parse the JSON data
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON response body: %s ; %v", string(body), err)
	}

	// Access the JSON data
	return data.([]interface{}), nil
}

// downloadFile downloads file - used for downloading individual patches.
func downloadFile(filePath string, parentPath string, url string) error {
	// Create parent directories if they don't exist
	err := os.MkdirAll(parentPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating parent directories: %v", err)
	}

	// Open the file in append mode or create it if it doesn't exist
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Get the current file size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}
	currentSize := fileInfo.Size()

	// Seek to the end of the file
	_, err = file.Seek(currentSize, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error seeking file: %v", err)
	}

	writer := bufio.NewWriter(file)

	err = getFileContentsFromUrl(url, currentSize, writer)
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

// getFileContentsFromUrl retrieves file contents stream from file at given url address
func getFileContentsFromUrl(url string, startSize int64, out *bufio.Writer) error {
	var err error
	var written int64

	for tries := maxNumberOfDownloadAttempts; tries > 0; tries-- {
		written, err = downloadFileContents(url, startSize, out)
		if err == nil {
			return nil
		}

		// wait until next attempt
		time.Sleep(1 * time.Second)

		startSize += written
	}

	return fmt.Errorf("failed after %v attempts; %s", maxNumberOfDownloadAttempts, err.Error())
}

// downloadFileContents downloads file contents from given start
func downloadFileContents(url string, startSize int64, out *bufio.Writer) (int64, error) {
	// Set the "Range" header to resume the download from the current size
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("error creating request: %v", err)
	}
	if startSize > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(startSize, 10)+"-")
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check server response again
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("downloading %s, bad status: %s", url, resp.Status)
	}

	// Writer the body to file
	return io.Copy(out, resp.Body)
}

// extractTarGz extracts tar file contents into location of output folder
func extractTarGz(tarGzFile, outputFolder string) error {
	// Open the tar.gz file
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create the gzip readerÏ
	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	// Create the tar reader
	tr := tar.NewReader(gr)

	// Extract the files from the tar reader
	for {
		header, err := tr.Next()
		if err == io.EOF {
			// Reached the end of the tar archive
			break
		}
		if err != nil {
			return err
		}

		// Determine the output file path
		targetPath := filepath.Join(outputFolder, header.Name)

		// Check if it's a directory
		if header.FileInfo().IsDir() {
			// Create the directory
			err = os.MkdirAll(targetPath, 0755)
			if err != nil {
				return err
			}
		} else {
			// Create the parent directory of the file
			err = os.MkdirAll(filepath.Dir(targetPath), 0755)
			if err != nil {
				return err
			}

			// Create the output file
			file, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			// Copy the file content from the tar reader to the output file
			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}

			err = file.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

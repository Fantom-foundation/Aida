// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package utildb

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/Fantom-foundation/Substate/db"
)

const (
	maxNumberOfDownloadAttempts = 5
	firstMainnetPatchFileName   = "5577-46750.tar.gz"
	firstTestnetPatchFileName   = "" // todo fill with first testnet patch once lachesis patch for testnet is released
	stateHashPatchFileName      = "state-hashes_0-68940000"
)

// Update implements updating command to be called from various commands and automatically downloads aida-db patches.
func Update(cfg *utils.Config) error {
	log := logger.NewLogger(cfg.LogLevel, "DB Update")
	start := time.Now()

	targetDbFirstBlock, targetDbLastBlock, err := getTargetDbBlockRange(cfg)
	if err != nil {
		return fmt.Errorf("unable retrieve aida-db metadata; %v", err)
	}

	log.Noticef("First block of your AidaDb: #%v", targetDbFirstBlock)
	log.Noticef("Last block of your AidaDb: #%v", targetDbLastBlock)

	// retrieve available patches from aida-db generation server
	patches, err := retrievePatchesToDownload(cfg, targetDbFirstBlock, targetDbLastBlock)
	if err != nil {
		return fmt.Errorf("unable to prepare list of aida-db patches for download; %v", err)
	}

	if len(patches) == 0 {
		log.Warning("No new patches to download are available")
		return nil
	}

	var str string
	for _, p := range patches {
		str += " "
		str += p.FileName
	}

	log.Noticef("These patches are in que for download:%v", str)

	// we need to know whether Db is new for metadata
	err = patchesDownloader(cfg, patches, targetDbFirstBlock, targetDbLastBlock)
	if err != nil {
		return err
	}

	log.Noticef("Aida-Db %v updated finished successfully. Total elapsed time: %v", cfg.AidaDb, time.Since(start).Round(1*time.Second))

	return nil
}

// getTargetDbBlockRange initialize aidaMetadata of targetDB
func getTargetDbBlockRange(cfg *utils.Config) (uint64, uint64, error) {
	// load stats of current aida-db to download just latest patches
	_, err := os.Stat(cfg.AidaDb)
	if err != nil {
		if os.IsNotExist(err) {
			// aida-db does not exist, download all available patches
			return 0, 0, nil
		} else {
			return 0, 0, err
		}
	} else {
		// load last block from existing aida-db metadata
		sdb, err := db.NewReadOnlySubstateDB(cfg.AidaDb)
		if err != nil {
			return 0, 0, err
		}
		defer sdb.Close()

		firstAidaDbBlock, lastAidaDbBlock, ok := utils.FindBlockRangeInSubstate(sdb)
		if !ok {
			return 0, 0, fmt.Errorf("cannot find blocks in substate; is substate present in given db? %v", cfg.AidaDb)
		}
		return firstAidaDbBlock, lastAidaDbBlock, nil
	}
}

// patchesDownloader processes patch names to download then download them in pipelined process
func patchesDownloader(cfg *utils.Config, patches []utils.PatchJson, firstBlock, lastBlock uint64) error {
	// create channel to push patch labels trough channel
	patchesChan := pushPatchToChanel(patches)

	// download patches
	downloadedPatchChan, errChan := downloadPatch(cfg, patchesChan)

	// decompress downloaded patches
	decompressedPatchChan, errDecompressChan := decompressPatch(cfg, downloadedPatchChan, errChan)

	// merge decompressed patches
	err := mergePatch(cfg, decompressedPatchChan, errDecompressChan, firstBlock, lastBlock)
	if err != nil {
		return err
	}

	return nil
}

// mergePatch takes decompressed patches and merges them into aida-db
func mergePatch(cfg *utils.Config, decompressChan chan string, errChan chan error, firstAidaDbBlock, lastAidaDbBlock uint64) error {
	var (
		err                       error
		patchDb                   db.BaseDB
		targetMD                  *utils.AidaDbMetadata
		patchDbHash, targetDbHash []byte
		isNewDb                   bool
		log                       = logger.NewLogger(cfg.LogLevel, "aida-merge-patch")
	)

	if lastAidaDbBlock == 0 {
		isNewDb = true
	}

	firstRun := true

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
					if cfg.Validate {
						if patchDbHash == nil {
							log.Critical("DbHash not found in downloaded Patch - cannot perform validation. If you were missing only lachesis patch, this would be normal behaviour.")
						} else {
							log.Notice("Starting db-validation. This may take several hours...")
							targetDbHash, err = GenerateDbHash(targetMD.Db, cfg.LogLevel)
							if err != nil {
								return fmt.Errorf("cannot create DbHash of merged AidaDb; %v", err)
							}

							if cmp := bytes.Compare(patchDbHash, targetDbHash); cmp != 0 {
								log.Criticalf("db hashes are not same! \nPatch: %v; Calculated: %v", hex.EncodeToString(patchDbHash), hex.EncodeToString(targetDbHash))
							} else {
								log.Notice("Validation successful!")
								return targetMD.SetDbHash(patchDbHash)
							}
						}
					}

					return nil
				}

				// firstRun is triggered only when applying first patch
				// distinction is necessary because if targetDb was empty we can move patch directly into targetPath
				// before opening database for writing
				if firstRun {
					firstRun = false
					// first patch to empty database is moved to target right away
					// this way we can skip iteration and metadata inserts
					if isNewDb {
						log.Noticef("AIDA-DB was empty - directly saving first patch")
						// move extracted patch to target location - first attempting with os.Rename because it is fastest
						if err = os.Rename(extractedPatchPath, cfg.AidaDb); err != nil {
							// attempting with deep copy - needed when moving across different disks
							if err2 := utils.CopyDir(extractedPatchPath, cfg.AidaDb); err2 != nil {
								return fmt.Errorf("unable to move patch into aida-db target; %v (%v)", err2, err)
							}
						}
					}

					// open targetDB only after there is already first patch or any existing previous data
					targetDb, err := db.NewDefaultBaseDB(cfg.AidaDb)
					if err != nil {
						return fmt.Errorf("can't open aidaDb; %v", err)
					}
					targetMD = utils.NewAidaDbMetadata(targetDb, cfg.LogLevel)

					errOldAida := targetMD.UpdateMetadataInOldAidaDb(cfg.ChainID, firstAidaDbBlock, lastAidaDbBlock)
					if errOldAida != nil {
						log.Warningf("error UpdateMetadataInOldAidaDb; %v", errOldAida)
					}

					defer func() {
						if err = targetMD.Db.Close(); err != nil {
							log.Warningf("patchesDownloader: cannot close targetDb; %v", err)
						}
					}()

					// patch was already applied before opening targetDb hence we don't need to merge it anymore
					if isNewDb {
						continue
					}
				}

				// merge newly extracted patch
				patchDb, err = db.NewReadOnlyBaseDB(extractedPatchPath)
				if err != nil {
					return fmt.Errorf("cannot open targetDb; %v", err)
				}

				// we only check metadata if not applying stateHashPatch
				if !strings.Contains(extractedPatchPath, stateHashPatchFileName) {
					// save patch dbHash - last hash gets validated if validation is turned on
					patchDbHash, err = targetMD.CheckUpdateMetadata(cfg, patchDb)
					if err != nil {
						return err
					}
				}

				m := NewMerger(cfg, targetMD.Db, []db.BaseDB{patchDb}, []string{extractedPatchPath}, nil)

				err = m.Merge()
				if err != nil {
					return fmt.Errorf("unable to merge %v; %v", extractedPatchPath, err)
				}

				// we only set metadata if not applying stateHashPatch
				if strings.Contains(extractedPatchPath, stateHashPatchFileName) {
					err = targetMD.SetHasHashPatch()
					if err != nil {
						return fmt.Errorf("cannot set has-hash-patch; %v", err)
					}
				} else {
					err = targetMD.SetAll()
					if err != nil {
						return fmt.Errorf("cannot set metadata; %v", err)
					}
				}
				m.CloseSourceDbs()

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
func decompressPatch(cfg *utils.Config, patchChan chan utils.PatchJson, errChan chan error) (chan string, chan error) {
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
			case patch, ok := <-patchChan:
				{
					if !ok {
						return
					}
					log.Debugf("Decompressing %v...", patch.FileName)

					compressedPatchPath := filepath.Join(cfg.DbTmp, patch.FileName)
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
func downloadPatch(cfg *utils.Config, patchesChan chan utils.PatchJson) (chan utils.PatchJson, chan error) {
	log := logger.NewLogger(cfg.LogLevel, "Download patch")
	downloadedPatchChan := make(chan utils.PatchJson, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(downloadedPatchChan)
		defer close(errChan)
		for {
			patch, ok := <-patchesChan
			if !ok {
				return
			}
			log.Debugf("Downloading %s...", patch.FileName)
			patchUrl := utils.AidaDbRepositoryUrl + "/" + patch.FileName
			compressedPatchPath := filepath.Join(cfg.DbTmp, patch.FileName)
			err := downloadFile(compressedPatchPath, cfg.DbTmp, patchUrl)
			if err != nil {
				errChan <- fmt.Errorf("unable to download %s; %v", patchUrl, err)
				return
			}
			log.Debugf("Finished downloading %s!", patch.FileName)

			log.Debugf("Calculating %s md5...", patch.FileName)
			md5, err := calculateMD5Sum(compressedPatchPath)
			if err != nil {
				errChan <- fmt.Errorf("archive %v; unable to calculate md5sum; %v", patch.FileName, err)
				return
			}

			// Compare whether downloaded file matches expected md5
			if strings.Compare(md5, patch.TarHash) != 0 {
				errChan <- fmt.Errorf("archive %v doesn't have matching md5; archive %v, expected %v", patch.FileName, md5, patch.TarHash)
				return
			}

			downloadedPatchChan <- patch
		}
	}()
	return downloadedPatchChan, errChan
}

// pushPatchToChanel used to pipe strings into channel
func pushPatchToChanel(strings []utils.PatchJson) chan utils.PatchJson {
	ch := make(chan utils.PatchJson, 1)
	go func() {
		defer close(ch)
		for _, str := range strings {
			ch <- str
		}
	}()
	return ch
}

// retrievePatchesToDownload retrieves all available patches from aida-db generation server.
func retrievePatchesToDownload(cfg *utils.Config, targetDbFirstBlock uint64, targetDbLastBlock uint64) ([]utils.PatchJson, error) {
	if cfg.UpdateType != "stable" && cfg.UpdateType != "nightly" {
		return nil, fmt.Errorf("please choose correct data-type with --data-type flag (stable/nightly)")
	}

	var includeNightly = cfg.UpdateType == "nightly"

	var isAddingLachesisPatch = false

	// download list of available availablePatches
	availablePatches, err := utils.DownloadPatchesJson()
	if err != nil {
		return nil, fmt.Errorf("unable to download patches.json: %v", err)
	}

	hasStateHashPatch, err := utils.HasStateHashPatch(cfg.AidaDb)
	if err != nil {
		return nil, err
	}

	// list of availablePatches to be downloaded
	var patchesToDownload = make([]utils.PatchJson, 0)

	for _, patch := range availablePatches {
		if patch.FileName == stateHashPatchFileName+".tar.gz" {
			if !hasStateHashPatch {
				patchesToDownload = append(patchesToDownload, patch)
			}
		}
		// skip every patch which is sooner than previous last block
		if patch.ToBlock <= targetDbLastBlock {
			// if patch is lachesis and user has not got it in their db we download it
			if patch.ToBlock == utils.FirstOperaBlock-1 && targetDbFirstBlock == utils.FirstOperaBlock {
				isAddingLachesisPatch = true
			} else {
				continue
			}
		}

		// add all stable patches and nightly only if user wants to download nightly
		if !patch.Nightly || includeNightly {
			patchesToDownload = append(patchesToDownload, patch)
		}
	}

	// if user has second patch already in their db, we have to re-download it again and delete old update-set key
	if isAddingLachesisPatch && targetDbFirstBlock == utils.FirstOperaBlock {
		patchesToDownload, err = appendFirstPatch(cfg, availablePatches, patchesToDownload)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(ByToBlock(patchesToDownload))

	return patchesToDownload, nil
}

// ByToBlock is an interface that is used to sort the patches by ToBlock
type ByToBlock []utils.PatchJson

func (a ByToBlock) Len() int {
	return len(a)
}
func (a ByToBlock) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByToBlock) Less(i, j int) bool {
	return a[i].ToBlock < a[j].ToBlock
}

// appendFirstPatch finds whether user is downloading fresh new db or updating an existing one.
// If updating an existing one, first patch is appended to download and first update-set is deleted
func appendFirstPatch(cfg *utils.Config, availablePatches []utils.PatchJson, patchesToDownload []utils.PatchJson) ([]utils.PatchJson, error) {
	var expectedFileName string

	if cfg.ChainID == utils.MainnetChainID {
		expectedFileName = firstMainnetPatchFileName
	} else if cfg.ChainID == utils.TestnetChainID {
		expectedFileName = firstTestnetPatchFileName
	} else {
		return nil, errors.New("please choose chain-id with --chainid")
	}

	// did we already append first patch?
	for _, patch := range patchesToDownload {
		if patch.FileName == expectedFileName {

			// first patch was already appended - that means user is downloading fresh db
			return patchesToDownload, nil
		}
	}

	for _, patch := range availablePatches {
		if patch.FileName == expectedFileName {
			patchesToDownload = append(patchesToDownload, patch)
			// we need to remove first update-set for data consistency
			err := deleteOperaWorldStateFromUpdateSet(cfg.AidaDb)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	return patchesToDownload, nil
}

// deleteOperaWorldStateFromUpdateSet when user has already merged second patch, and we are prepending lachesis patch.
// This situation could happen due to lachesis patch being implemented later than rest of the Db
func deleteOperaWorldStateFromUpdateSet(dbPath string) error {
	updateDb, err := db.NewDefaultUpdateDB(dbPath)
	if err != nil {
		return fmt.Errorf("cannot open update-db; %v", err)
	}

	err = updateDb.DeleteUpdateSet(utils.FirstOperaBlock - 1)
	if err != nil {
		return err
	}

	return updateDb.Close()
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
		time.Sleep(2 * time.Second)

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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
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
		targetPath, err := filepath.Abs(filepath.Join(outputFolder, header.Name))
		if err != nil {
			return err
		}

		// Make sure that path does not contain ".."
		if strings.Contains(targetPath, "..") {
			return fmt.Errorf("Tarfile is attempting to use path containing ..: %s", targetPath)
		}

		// Make sure that output file does not overwrite existing files
		_, err = os.Stat(targetPath)
		if err != nil && errors.Is(err, os.ErrNotExist) { // check if not exist is clearer than check if exists
			continue
		} else {
			return fmt.Errorf("Tarfile is attempting to overwrite existing file: %s", targetPath)
		}

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

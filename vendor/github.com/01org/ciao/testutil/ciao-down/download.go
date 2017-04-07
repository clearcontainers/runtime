/*
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

type downloadInfo struct {
	imageName    string
	imageTmpName string
}

type progressCB func(p progress)

type progress struct {
	downloadedMB int
	totalMB      int
}

type progressReader struct {
	downloaded int64
	totalMB    int
	reader     io.Reader
	cb         progressCB
}

func (pr *progressReader) Read(p []byte) (int, error) {
	read, err := pr.reader.Read(p)
	if err == nil {
		oldMB := pr.downloaded / 10000000
		pr.downloaded += int64(read)
		newMB := pr.downloaded / 10000000
		if newMB > oldMB {
			pr.cb(progress{downloadedMB: int(newMB * 10), totalMB: pr.totalMB})
		}
	}
	return read, err
}

func makeDownloadInfo(URL string) (downloadInfo, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return downloadInfo{}, fmt.Errorf("unable to parse %s:%v", err, URL)
	}
	di := downloadInfo{
		imageName: filepath.Base(u.Path),
	}
	di.imageTmpName = di.imageName + ".part"
	return di, nil
}

func getFile(ctx context.Context, URL string, dest io.WriteCloser, cb progressCB) (err error) {
	defer func() {
		err1 := dest.Close()
		if err == nil && err1 != nil {
			err = err1
		}
	}()
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	cli := &http.Client{Transport: http.DefaultTransport}
	resp, err := cli.Do(req)
	if err != nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Failed to download %s : %s", URL, resp.Status)
		return
	}

	pr := &progressReader{reader: resp.Body, cb: cb}
	if resp.ContentLength == -1 {
		pr.totalMB = -1
	} else {
		pr.totalMB = int(resp.ContentLength / 1000000)
	}

	buf := make([]byte, 1<<20)
	_, err = io.CopyBuffer(dest, pr, buf)

	if err == nil && int(pr.downloaded/1000000)%10 != 0 {
		cb(progress{downloadedMB: pr.totalMB, totalMB: pr.totalMB})
	}

	return
}

func downloadProgress(p progress) {
	if p.totalMB >= 0 {
		fmt.Printf("Downloaded %d MB of %d\n", p.downloadedMB, p.totalMB)
	} else {
		fmt.Printf("Downloaded %d MB\n", p.downloadedMB)
	}
}

func downloadFile(ctx context.Context, URL, ciaoDir string, cb progressCB) (string, error) {
	di, err := makeDownloadInfo(URL)
	if err != nil {
		return "", err
	}

	cacheDir := path.Join(ciaoDir, "cache")
	imgPath := path.Join(cacheDir, di.imageName)

	if _, err := os.Stat(imgPath); err == nil {
		return imgPath, nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("Unable to create directory %s : %v",
			cacheDir, err)
	}

	// Handles legacy code in which the ubuntu image used to be stored in
	// the root of the ~/.ciao_down directory.  We don't want to move the
	// old image as this would break any existing VMs based off it.

	oldImgPath := path.Join(ciaoDir, di.imageName)
	if err := exec.Command("cp", oldImgPath, imgPath).Run(); err == nil {
		return imgPath, nil
	}

	tmpImgPath := path.Join(cacheDir, di.imageTmpName)

	if _, err := os.Stat(imgPath); err == nil {
		return imgPath, nil
	}

	if _, err := os.Stat(tmpImgPath); err == nil {
		_ = os.Remove(tmpImgPath)
	}

	f, err := os.Create(tmpImgPath)
	if err != nil {
		return "", fmt.Errorf("Unable to create download file: %v", err)
	}

	err = getFile(ctx, URL, f, cb)
	if err != nil {
		_ = os.Remove(tmpImgPath)
		return "", fmt.Errorf("Unable download file %s: %v",
			URL, err)
	}

	err = os.Rename(tmpImgPath, imgPath)
	if err != nil {
		_ = os.Remove(tmpImgPath)
		return "", fmt.Errorf("Unable move downloaded file to %s: %v",
			imgPath, err)
	}

	return imgPath, nil
}

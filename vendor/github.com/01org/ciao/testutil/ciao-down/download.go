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
	"os"
	"path"
)

// Constants for the Guest image used by ciao-down

const (
	guestDownloadURL       = "https://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-disk1.img"
	guestImageName         = "xenial-server-cloudimg-amd64-disk1.img"
	guestImageTmpName      = "xenial-server-cloudimg-amd64-disk1.img.part"
	guestImageFriendlyName = "Ubuntu 16.04"
)

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

func getUbuntu(ctx context.Context, dest io.WriteCloser, cb progressCB) (err error) {
	defer func() {
		err1 := dest.Close()
		if err == nil && err1 != nil {
			err = err1
		}
	}()
	req, err := http.NewRequest("GET", guestDownloadURL, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	cli := &http.Client{Transport: http.DefaultTransport}
	resp, err := cli.Do(req)
	if err != nil {
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
	_ = resp.Body.Close()

	if err == nil && int(pr.downloaded/1000000)%10 != 0 {
		downloadProgress(progress{downloadedMB: pr.totalMB, totalMB: pr.totalMB})
	}

	return
}

func downloadUbuntu(ctx context.Context, instanceDir string, cb progressCB) (string, error) {
	imgPath := path.Join(instanceDir, guestImageName)

	if _, err := os.Stat(imgPath); err == nil {
		return imgPath, nil
	}

	fmt.Printf("Downloading %s\n", guestImageFriendlyName)

	tmpImgPath := path.Join(instanceDir, guestImageTmpName)

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

	err = getUbuntu(ctx, f, cb)
	if err != nil {
		_ = os.Remove(tmpImgPath)
		return "", fmt.Errorf("Unable download file %s: %v",
			guestDownloadURL, err)
	}

	err = os.Rename(tmpImgPath, imgPath)
	if err != nil {
		_ = os.Remove(tmpImgPath)
		return "", fmt.Errorf("Unable move downloaded file to %s: %v",
			imgPath, err)
	}

	return imgPath, nil
}

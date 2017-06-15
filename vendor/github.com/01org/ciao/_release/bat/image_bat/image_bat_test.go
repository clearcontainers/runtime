//
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
//

package imagebat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/01org/ciao/bat"
	"github.com/01org/ciao/ssntp/uuid"
)

const standardTimeout = time.Second * 300

// addShowDelete adds a new image containing random content to the image
// service.  It then retrieves the meta data for the new image and checks that
// various fields are correct.  Finally, it deletes the image. It will also
// receive the visibility in order images are created with the right visibility
//
// The image is successfully uploaded, it appears when ciao-cli image show is
// executed, it can be successfully deleted and is no longer present in the
// ciao-cli image list output after deletion.
func addShowDelete(t *testing.T, visibility string) {
	name := fmt.Sprintf("test-image-%v", visibility)
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()

	// TODO:  The only options currently supported by the image service are
	// ID, Visibility and  Name. This code needs to be updated when the image service's
	// support for meta data improves.

	options := bat.ImageOptions{
		Name:       name,
		Visibility: visibility,
	}
	img, err := bat.AddRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	if img.ID == "" || img.Name != name || img.Status != "active" ||
		img.Visibility != visibility || img.Protected {
		t.Errorf("Meta data of added image is incorrect")
	}

	if img.ID != "" {
		gotImg, err := bat.GetImage(ctx, "", img.ID)
		if err != nil {
			t.Errorf("Unable to retrieve meta data for image %v", err)
		} else if gotImg.ID != img.ID || gotImg.Name != img.Name {
			t.Errorf("Unexpected meta data retrieved for image")
		}

		err = bat.DeleteImage(ctx, "", img.ID)
		if err != nil {
			t.Fatalf("Unable to delete image %v", err)
		}

		_, err = bat.GetImage(ctx, "", img.ID)
		if err == nil {
			t.Fatalf("Call to get non-existing image should fail")
		}
	}
}

// TestPrivateAddShowDelete adds a new private image, checks it's listed and deletes it
func TestPrivateAddShowDelete(t *testing.T) {
	addShowDelete(t, "private")
}

// TestPublicAddShowDelete adds a new public image, checks it's listed and deletes it
func TestPublicAddShowDelete(t *testing.T) {
	addShowDelete(t, "public")
}

// TestInternalAddShowDelete adds a new internal image, checks it's listed and deletes it
func TestInternalAddShowDelete(t *testing.T) {
	addShowDelete(t, "internal")
}

// Delete a non-existing image
//
// TestDeleteNonExisting attempts to delete an non-existing image.
//
// The attempt to delete the non-existing image should fail
func TestDeleteNonExisting(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	err := bat.DeleteImage(ctx, "", uuid.Generate().String())
	cancelFunc()
	if err == nil {
		t.Errorf("Call to delete non-existing image should fail")
	}
}

// Check image list works correctly
//
// TestImageList retrieves the number of images in the image service, adds a new
// image, retrieves the image list once more, and then deletes the newly added image.
//
// The meta data received for each image should be correct, the meta data for the
// image should be present and the list of images returned by the image service
// should increase by 1 after the image has been added.  The image should be
// destroyed without error.
func TestImageList(t *testing.T) {
	const name = "test-image"
	ctx, cancelFunc := context.WithTimeout(context.Background(), standardTimeout)
	defer cancelFunc()
	count, err := bat.GetImageCount(ctx, "")
	if err != nil {
		t.Fatalf("Unable to count number of images: %v", err)
	}

	options := bat.ImageOptions{
		Name: name,
	}
	img, err := bat.AddRandomImage(ctx, "", 10, &options)
	if err != nil {
		t.Fatalf("Unable to add image %v", err)
	}

	images, err := bat.GetImages(ctx, "")
	if err != nil {
		t.Errorf("Unable to retrieve image list: %v", err)
	}

	if len(images) != count+1 {
		t.Errorf("Unexpected number of images, expected %d got %d", count+1, len(images))
	}

	foundNewImage := false
	for k, newImg := range images {
		foundNewImage = k == img.ID
		if foundNewImage {
			if newImg.ID == "" || newImg.Name != name || newImg.Status != "active" ||
				newImg.Visibility != "private" || newImg.Protected {
				t.Errorf("Meta data of added image is incorrect")
			}
			break
		}
	}

	if !foundNewImage {
		t.Errorf("New image was not returned by ciao-cli image list")
	}

	err = bat.DeleteImage(ctx, "", img.ID)
	if err != nil {
		t.Fatalf("Unable to delete image %v", err)
	}
}

/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake_test

import (
	"fmt"
	"path/filepath"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var fsys fake.OSBuilder

var _ = Describe("Mockfs", func() {
	var err error
	JustBeforeEach(func() {
		os, err := fake.NewOS("/")
		fsys = fake.BuilderForOS(os)
		Expect(err).NotTo(HaveOccurred())
	})

	const dirName = "dir"

	whenDirSuccessfullyCreated(dirName, func() {
		whenCallingMkdir(dirName, func(err *error) {
			It("fails to create existing directory", func() {
				Expect(*err).To(HaveOccurred())
			})
		})

		whenCallingFileCreate(dirName, func(_ *fs.File, err *error) {
			It("fails to create file with existing directory name", func() {
				Expect(*err).To(HaveOccurred())
			})
		})

		fileInDirPath := filepath.Join(dirName, "file.txt")

		whenFileSuccessfullyCreated(fileInDirPath, func(fptr *fs.File) {
			whenCallingMkdir(fileInDirPath, func(err *error) {
				It("fails to create directory with existing file name", func() {
					Expect(*err).To(HaveOccurred())
				})
			})

			whenCallingFileCreate(fileInDirPath, func(_ *fs.File, err *error) {
				It("fails to create file with existing file name", func() {
					Expect(*err).To(HaveOccurred())
				})
			})

			It("get file should return correct file", func() {
				By("checking is not directory")
				fileObj, err := fsys.GetFile(fileInDirPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(fileObj).NotTo(BeNil())
				Expect(fileObj.Mode().IsDir()).To(BeFalse(), "Created file should not be directory")
				Expect(fileObj.Mode().IsRegular()).To(BeTrue())
			})
		})

		itFailsToCreateFileInMissingDirectory()
	})

	itFailsToCreateFileInMissingDirectory()
	whenFileSuccessfullyCreated("file.txt", func(_ *fs.File) {
		It("failed to create file in file", func() {
			_, err = fsys.Create("file.txt/child.txt")
			Expect(err).To(HaveOccurred())
		})

		itFailsToCreateFileInMissingDirectory()
	})
})

func itFailsToCreateFileInMissingDirectory() {
	It("fails to create file in missing directory", func() {
		_, err := fsys.Create("missing/file.txt")
		Expect(err).To(HaveOccurred())
	})
}

func whenCallingMkdir(dirName string, fn func(*error)) {
	When(fmt.Sprintf("directory '%s' created", dirName), func() {
		var err error
		JustBeforeEach(func() {
			err = fsys.Mkdir(dirName, 0o755)
		})

		fn(&err)
	})
}

func whenDirSuccessfullyCreated(dirName string, fn func()) {
	whenCallingMkdir(dirName, func(err *error) {
		JustBeforeEach(func() {
			Expect(*err).NotTo(HaveOccurred())
		})

		It("succeed", func() {
			// in case we don't have it in fn
		})

		fn()
	})
}

func whenCallingFileCreate(name string, fn func(*fs.File, *error)) {
	When(fmt.Sprintf("directory '%s' created", name), func() {
		var err error
		var file fs.File
		JustBeforeEach(func() {
			file, err = fsys.Create(name)
		})

		fn(&file, &err)
	})
}

func whenFileSuccessfullyCreated(dirName string, fn func(file *fs.File)) {
	whenCallingFileCreate(dirName, func(fptr *fs.File, err *error) {
		JustBeforeEach(func() {
			Expect(*err).NotTo(HaveOccurred())
			Expect(*fptr).NotTo(BeNil(), "file is not nil")
		})

		It("succeed", func() {
			// in case we don't have it in fn
		})

		fn(fptr)
	})
}

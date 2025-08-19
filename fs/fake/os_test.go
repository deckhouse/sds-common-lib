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
	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OS", func() {
	rootPath := "/"
	var builder *fake.Builder
	JustBeforeEach(func() {
		builder = fake.NewBuilder(rootPath)
	})
	When("built", func() {
		var os *fake.OS
		var buildErr error

		JustBeforeEach(func() {
			os, buildErr = builder.
				WithFile("file").
				WithFile("dir", fake.NewFile("child")).
				WithFile("link", fake.LinkReader{Target: "./dir/child"}).
				Build()
			Expect(buildErr).ToNot(HaveOccurred())
		})

		It("Chmod not existing", func() {
			err := os.Chmod("file1", fs.FileMode(0o123))
			Expect(err).To(HaveOccurred())
		})

		It("Chmod", func() {
			err := os.Chmod("file", fs.FileMode(0o123))
			Expect(err).ToNot(HaveOccurred())

			fi, err := os.Stat("file")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.Mode() & fs.ModePerm).To(BeEquivalentTo(0o123))
		})

		It("Chown", func() {
			err := os.Chown("file", 1, 2)
			Expect(err).ToNot(HaveOccurred())

			// TODO: check if set
			// fi, err := os.Stat("file")
			// Expect(err).ToNot(HaveOccurred())
			// Expect(fi.Sys(). & fs.ModePerm).To(BeEquivalentTo(0o123))
		})

		It("Chown not existing", func() {
			err := os.Chown("file1", 1, 2)
			Expect(err).To(HaveOccurred())
		})

		It("DirFS", func() {
			fs := os.DirFS("/")
			Expect(fs).ToNot(BeNil())

			_, err := fs.Open("file")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Open("link")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Open("file1")
			Expect(err).To(HaveOccurred())
		})

		It("Open", func() {
			file, err := os.Open("file")
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Open link", func() {
			file, err := os.Open("link")
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Open not existing", func() {
			file, err := os.Open("file1")
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Create", func() {
			file, err := os.Create("file1")
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())

			openedFile, err := os.Open("file1")
			Expect(openedFile).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Create existing", func() {
			file, err := os.Create("file")
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("Create existing link", func() {
			file, err := os.Create("link")
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("OpenFile existing", func() {
			file, err := os.OpenFile("file", 0, 0)
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("OpenFile existing link", func() {
			file, err := os.OpenFile("link", 0, 0)
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("OpenFile not existing", func() {
			file, err := os.OpenFile("file1", 0, 0)
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("OpenFile existing create", func() {
			file, err := os.OpenFile("file", fs.O_CREATE, 0)
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("OpenFile existing link create", func() {
			file, err := os.OpenFile("link", fs.O_CREATE, 0)
			Expect(file).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("OpenFile not existing create", func() {
			file, err := os.OpenFile("file1", fs.O_CREATE, 0)
			Expect(file).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
		})

		It("Stat", func() {
			fi, err := os.Stat("file")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi).ToNot(BeNil())

			Expect(fi.Name()).To(BeEquivalentTo("file"))
		})

		It("Stat link", func() {
			fi, err := os.Stat("link")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi).ToNot(BeNil())

			Expect(fi.Name()).To(BeEquivalentTo("child"))
		})

		It("Lstat", func() {
			fi, err := os.Lstat("file")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi).ToNot(BeNil())

			Expect(fi.Name()).To(BeEquivalentTo("file"))
		})

		It("Lstat link", func() {
			fi, err := os.Lstat("link")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi).ToNot(BeNil())

			Expect(fi.Name()).To(BeEquivalentTo("link"))
		})

		It("ReadDir", func() {
			list, err := os.ReadDir("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(3))
		})

		It("ReadDir file", func() {
			_, err := os.ReadDir("file")
			Expect(err).To(HaveOccurred())
		})

		It("Chdir", func() {
			Expect(os.Chdir("dir")).To(Succeed())
			Expect(os.Getwd()).To(Equal("/dir"))

			Expect(os.Chdir("..")).To(Succeed())
			Expect(os.Getwd()).To(Equal("/"))
		})

		It("Getwd", func() {
			Expect(os.Getwd()).To(Equal("/"))
		})

		It("Mkdir", func() {
			Expect(os.Mkdir("dir2", 0)).To(Succeed())
			fi, err := os.Stat("dir2")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.IsDir()).To(BeTrue())
			Expect(os.ReadDir("/")).To(HaveLen(4))
			Expect(os.ReadDir("dir2")).To(HaveLen(0))
		})

		It("Mkdir abs", func() {
			Expect(os.Mkdir("/dir2", 0)).To(Succeed())
			fi, err := os.Stat("dir2")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.IsDir()).To(BeTrue())
			Expect(os.ReadDir("/")).To(HaveLen(4))
			Expect(os.ReadDir("dir2")).To(HaveLen(0))
		})

		It("Mkdir in dir", func() {
			Expect(os.Mkdir("dir/dir2", 0)).To(Succeed())
			fi, err := os.Stat("dir/dir2")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.IsDir()).To(BeTrue())

			Expect(os.ReadDir("/")).To(HaveLen(3))
			Expect(os.ReadDir("dir")).To(HaveLen(2))
		})

		It("Mkdir in not existing dir", func() {
			Expect(os.Mkdir("dir1/dir2", 0)).ToNot(Succeed())

			Expect(os.ReadDir("/")).To(HaveLen(3))
			Expect(os.ReadDir("dir")).To(HaveLen(1))
		})

		It("MkdirAll", func() {
			Expect(os.MkdirAll("dir2", 0)).To(Succeed())
		})

		It("MkdirAll in dir", func() {
			Expect(os.MkdirAll("dir/dir2", 0)).To(Succeed())
		})

		It("MkdirAll in not existing dir", func() {
			Expect(os.MkdirAll("dir1/dir2", 0)).To(Succeed())
		})

		It("MkdirAll in file", func() {
			Expect(os.MkdirAll("file/dir2", 0)).ToNot(Succeed())
		})

		It("Symlink", func() {
			Expect(os.Symlink("file", "newLink")).To(Succeed())
			file, err := os.Open("newLink")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
		})

		It("Symlink of dir", func() {
			Expect(os.Symlink("dir", "newLink")).To(Succeed())
			file, err := os.Open("newLink")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
		})

		It("Symlink on top of existing file", func() {
			Expect(os.Symlink("dir", "file")).ToNot(Succeed())
		})

		It("Symlink on top of existing directory", func() {
			Expect(os.Symlink("file", "dir")).ToNot(Succeed())
		})

		It("Symlink in non existing directory", func() {
			Expect(os.Symlink("file", "dir1/file")).ToNot(Succeed())
		})

		It("ReadLink", func() {
			Expect(os.ReadLink("link")).To(Equal("./dir/child"))
		})
	})
})

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
	"strings"

	"github.com/deckhouse/sds-common-lib/fs/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("builder", func() {
	var builder *fake.Builder
	JustBeforeEach(func() {
		builder = fake.NewBuilder("/")
		Expect(builder).ShouldNot(BeNil())
	})

	It("has root path", func() {
		Expect(builder.Root().Path()).To(Equal("/"))
		Expect(builder.GetWdFile().Path()).To(Equal("/"))
		Expect(builder.Root().Path()).To(Equal("/"))
	})

	When("built", func() {
		var builderOS *fake.OS
		var os *fake.OS
		var err error
		JustBeforeEach(func() {
			builderOS = builder.OS
			Expect(builderOS).ToNot(BeNil())
			os, err = builder.Build()
		})
		It("pass ownership of os", func() {
			Expect(os).To(BeIdenticalTo(builderOS))
			Expect(err).ToNot(HaveOccurred())
			Expect(builder.OS).To(BeNil())
		})
	})

	When("simple tree created using WithFile", func() {
		JustBeforeEach(func() {
			builder.
				WithFile("file").
				WithFile("dir",
					fake.NewFile("child"))
		})

		It("dir to be correct", func() {
			dir := builder.Root().GetChild("dir")
			Expect(dir).NotTo(BeNil())
			Expect(dir.Path()).To(BeEquivalentTo("/dir"))
			Expect(dir.Mode().IsDir()).To(BeTrue())
			Expect(dir.Mode().IsRegular()).To(BeFalse())
		})
		It("child to be correct", func() {
			dir := builder.Root().GetChild("dir")
			Expect(dir).NotTo(BeNil())

			child := dir.GetChild("child")
			Expect(child).NotTo(BeNil())
			Expect(child.Path()).To(BeEquivalentTo("/dir/child"))
			Expect(child.Mode().IsDir()).To(BeFalse())
			Expect(child.Mode().IsRegular()).To(BeTrue())
		})
		It("file to be correct", func() {
			file := builder.Root().GetChild("file")
			Expect(file).NotTo(BeNil())
			Expect(file.Path()).To(BeEquivalentTo("/file"))
			Expect(file.Mode().IsDir()).To(BeFalse())
			Expect(file.Mode().IsRegular()).To(BeTrue())
		})
		It("should not find not existing files", func() {
			Expect(builder.Root().GetChild("file1")).To(BeNil())

			dir := builder.Root().GetChild("dir")
			Expect(dir).NotTo(BeNil())
			Expect(dir.GetChild("dir")).To(BeNil())
		})

		It("can GetEntry by full path", func() {
			Expect(builder.GetEntry("/dir/child")).ToNot(BeNil())
		})
		It("can GetEntry by relative path", func() {
			Expect(builder.GetEntry("dir/child")).ToNot(BeNil())
			Expect(builder.GetEntry("file")).ToNot(BeNil())
			Expect(builder.GetEntry("dir/../file")).ToNot(BeNil())
			Expect(builder.GetEntry("./dir/child")).ToNot(BeNil())
			Expect(builder.GetEntry("./file")).ToNot(BeNil())
			Expect(builder.GetEntry("./dir/../file")).ToNot(BeNil())
		})

		When("WdFile is set to dir", func() {
			JustBeforeEach(func() {
				builder.SetWdFile(builder.Root().GetChild("dir"))
			})
			It("has correct wd file", func() {
				wd := builder.GetWdFile()
				Expect(wd).ToNot(BeNil())
				Expect(wd).To(BeEquivalentTo(builder.Root().GetChild("dir")))
			})
			It("can GetEntry by full path", func() {
				Expect(builder.GetEntry("/dir/child")).ToNot(BeNil())
				Expect(builder.GetEntry("/file")).ToNot(BeNil())
			})
			It("can GetEntry by relative path", func() {
				Expect(builder.GetEntry("child")).ToNot(BeNil())
				Expect(builder.GetEntry("../file")).ToNot(BeNil())
				Expect(builder.GetEntry("../dir/child")).ToNot(BeNil())
				Expect(builder.GetEntry("./child")).ToNot(BeNil())
				Expect(builder.GetEntry("./../file")).ToNot(BeNil())
				Expect(builder.GetEntry("./../dir/child")).ToNot(BeNil())
			})
			It("can not GetEntry by relative path as from root", func() {
				_, err := builder.GetEntry("dir/child")
				Expect(err).To(HaveOccurred())

				_, err = builder.GetEntry("file")
				Expect(err).To(HaveOccurred())
			})
			It("goes back if changed to Root", func() {
				builder.SetWdFile(builder.Root())
				Expect(builder.GetWdFile()).To(BeEquivalentTo(builder.Root()))
			})
		})

		When("same file added", func() {
			JustBeforeEach(func() {
				builder = builder.WithFile("file")
			})
			It("returns error on build", func() {
				os, err := builder.Build()
				Expect(err).To(HaveOccurred())
				Expect(os).ToNot(BeNil())
			})
		})

		It("indicate error if creating the same file", func() {
			entry, err := builder.CreateEntry("file")
			Expect(entry).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		When("built", func() {
			var os *fake.OS
			var err error
			JustBeforeEach(func() {
				os, err = builder.Build()

			})

			It("succeed", func() {
				Expect(os).ToNot(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})

			It("pass ownership of os", func() {
				Expect(builder.OS).To(BeNil())
			})

			It("dir to be correct", func() {
				dir, err := os.Open("dir")
				Expect(err).ToNot(HaveOccurred())
				Expect(dir).NotTo(BeNil())

				Expect(dir.Name()).To(BeEquivalentTo("dir"))
				stat, err := dir.Stat()
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Mode().IsDir()).To(BeTrue())
				Expect(stat.Mode().IsRegular()).To(BeFalse())
			})
			It("child to be correct", func() {
				child, err := os.Open("dir/child")
				Expect(err).ToNot(HaveOccurred())
				Expect(child).NotTo(BeNil())

				Expect(child.Name()).To(BeEquivalentTo("child"))
				stat, err := child.Stat()
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Mode().IsDir()).To(BeFalse())
				Expect(stat.Mode().IsRegular()).To(BeTrue())
			})
			It("file to be correct", func() {
				child, err := os.Open("file")
				Expect(err).ToNot(HaveOccurred())
				Expect(child).NotTo(BeNil())

				Expect(child.Name()).To(BeEquivalentTo("file"))
				stat, err := child.Stat()
				Expect(err).ToNot(HaveOccurred())

				Expect(stat.Mode().IsDir()).To(BeFalse())
				Expect(stat.Mode().IsRegular()).To(BeTrue())
			})
			It("should not find not existing files", func() {
				_, err := os.Open("file1")
				Expect(err).To(HaveOccurred())

				_, err = os.Open("dir/child1")
				Expect(err).To(HaveOccurred())
			})

			When("builder for created", func() {
				var builder fake.Builder
				JustBeforeEach(func() {
					builder = fake.BuilderFor(os)
				})

				It("can manage the os", func() {
					Expect(builder.Root().GetChild("file")).ToNot(BeNil())
					Expect(builder.Root().GetChild("dir")).ToNot(BeNil())
					Expect(builder.Root().GetChild("dir").GetChild("child")).ToNot(BeNil())
				})
			})
		})
	})
	It("make file with existing reader", func() {
		myString := "This is a sample string."
		reader := strings.NewReader(myString)

		_, err := builder.CreateEntry("file", reader)
		Expect(err).ToNot(HaveOccurred())

		file, err := builder.Open("file")
		Expect(err).ToNot(HaveOccurred())

		p := make([]byte, len(myString))
		n, err := file.Read(p)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(BeEquivalentTo(len(myString)))
	})

	Describe("WithFileAtPath", func() {
		It("should create a file with content at a nested path", func() {
			content := "test content"
			builder.WithFileAtPath("dir1/dir2/file.txt", fake.RWContentFromString(content))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Verify the file exists and has correct content
			file, err := fsys.Open("dir1/dir2/file.txt")
			Expect(err).ToNot(HaveOccurred())

			p := make([]byte, len(content))
			n, err := file.Read(p)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeEquivalentTo(len(content)))
			Expect(string(p)).To(Equal(content))
		})

		It("should create a directory at a nested path", func() {
			builder.WithFileAtPath("sys/block/dm-1/dm", fake.NewFile("dm"))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Verify the directory exists
			file, err := fsys.Open("sys/block/dm-1/dm")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
		})

		It("should create multiple files in the same directory structure", func() {
			builder.WithFileAtPath("etc/kubernetes/manifests/pod1.yaml", fake.RWContentFromString("apiVersion: v1"))
			builder.WithFileAtPath("etc/kubernetes/manifests/pod2.yaml", fake.RWContentFromString("apiVersion: v1"))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Verify both files exist
			file1, err := fsys.Open("etc/kubernetes/manifests/pod1.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(file1).ToNot(BeNil())

			file2, err := fsys.Open("etc/kubernetes/manifests/pod2.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(file2).ToNot(BeNil())
		})

		It("should create a file in the root directory", func() {
			builder.WithFileAtPath("rootfile.txt", fake.RWContentFromString("root content"))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			file, err := fsys.Open("rootfile.txt")
			Expect(err).ToNot(HaveOccurred())

			p := make([]byte, 12)
			n, err := file.Read(p)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(p[:n])).To(Equal("root content"))
		})

		It("should handle deep nested paths", func() {
			deepPath := "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/file.txt"
			content := "deep content"
			builder.WithFileAtPath(deepPath, fake.RWContentFromString(content))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			file, err := fsys.Open(deepPath)
			Expect(err).ToNot(HaveOccurred())

			p := make([]byte, len(content))
			n, err := file.Read(p)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(p[:n])).To(Equal(content))
		})

		It("should work with device mapper example", func() {
			// This is the exact use case from our scanner tests
			builder.WithFileAtPath("sys/block/dm-1/dm/name", fake.RWContentFromString("ubuntu--vg-ubuntu--lv"))
			builder.WithFileAtPath("dev/mapper/ubuntu--vg-ubuntu--lv", fake.NewFile("ubuntu--vg-ubuntu--lv"))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			// Verify the dm name file
			file, err := fsys.Open("sys/block/dm-1/dm/name")
			Expect(err).ToNot(HaveOccurred())

			p := make([]byte, 25)
			n, err := file.Read(p)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(p[:n])).To(Equal("ubuntu--vg-ubuntu--lv"))

			// Verify the mapper device
			mapperFile, err := fsys.Open("dev/mapper/ubuntu--vg-ubuntu--lv")
			Expect(err).ToNot(HaveOccurred())
			Expect(mapperFile).ToNot(BeNil())
		})

		It("should handle empty path gracefully", func() {
			// This should not panic or cause issues
			// Empty path should be treated as root directory
			builder.WithFileAtPath("", fake.RWContentFromString("test"))

			fsys, err := builder.Build()
			// We expect an error for empty path, but it should not panic
			Expect(err).To(HaveOccurred())
			Expect(fsys).ToNot(BeNil())
		})

		It("should handle single character paths", func() {
			builder.WithFileAtPath("a", fake.RWContentFromString("single"))

			fsys, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			file, err := fsys.Open("a")
			Expect(err).ToNot(HaveOccurred())

			p := make([]byte, 6)
			n, err := file.Read(p)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(p[:n])).To(Equal("single"))
		})
	})
})

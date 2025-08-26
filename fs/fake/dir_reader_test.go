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
	"io"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DirReader", func() {
	var b *fake.Builder
	JustBeforeEach(func() {
		b = fake.NewBuilder("/")
	})
	It("reads empty dir", func() {
		Expect(b.ReadDir("/")).To(BeEmpty())
	})
	When("opened", func() {
		var file fs.File
		JustBeforeEach(func() {
			var err error
			file, err = b.Open("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(file).ToNot(BeNil())
		})
		It("reads empty dir", func() {
			l, err := file.ReadDir(1)
			Expect(err).To(MatchError(io.EOF))
			Expect(l).To(BeEmpty())
		})
	})
	When("files are added", func() {
		names := []string{"file1", "file2", "file3"}
		JustBeforeEach(func() {
			for _, name := range names {
				Expect(b.CreateEntry(name)).ToNot(BeNil())
			}
		})
		It("reads the dir", func() {
			l, err := b.ReadDir("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(HaveLen(len(names)))
			receivedNames := make([]string, len(names))
			for i, li := range l {
				receivedNames[i] = li.Name()
			}
			Expect(receivedNames).To(ContainElements(names))
		})
		When("opened", func() {
			var file fs.File
			JustBeforeEach(func() {
				var err error
				file, err = b.Open("/")
				Expect(err).ToNot(HaveOccurred())
				Expect(file).ToNot(BeNil())
			})
			It("reads whole dir", func() {
				l, err := file.ReadDir(3)
				Expect(err).ToNot(HaveOccurred())
				Expect(l).To(HaveLen(len(names)))
				receivedNames := make([]string, len(names))
				for i, li := range l {
					receivedNames[i] = li.Name()
				}
				Expect(receivedNames).To(ContainElements(names))
			})
			It("reads dir by chunks", func() {
				l, err := file.ReadDir(2)
				Expect(err).ToNot(HaveOccurred())
				l2, err := file.ReadDir(2)
				Expect(err).ToNot(HaveOccurred())
				_, err = file.ReadDir(2)
				Expect(err).To(MatchError(io.EOF))

				l = append(l, l2...)

				Expect(l).To(HaveLen(len(names)))
				receivedNames := make([]string, len(names))
				for i, li := range l {
					receivedNames[i] = li.Name()
				}
				Expect(receivedNames).To(ContainElements(names))
			})
			When("closed", func() {
				JustBeforeEach(func() {
					Expect(file.Close()).To(Succeed())
				})
				It("can't close again", func() {
					Expect(file.Close()).To(MatchError(fs.ErrClosed))
				})
				It("can't list", func() {
					_, err := file.ReadDir(2)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})
		})
	})
})

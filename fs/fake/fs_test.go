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
	"github.com/deckhouse/sds-common-lib/fs/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FS", func() {
	var os *fake.OS
	JustBeforeEach(func() {
		var err error
		os, err = fake.NewBuilder("/", fake.NewFile("file"), fake.NewFile("dir", fake.NewFile("child"))).Build()
		Expect(err).ToNot(HaveOccurred())
	})
	It("opened from root", func() {
		fs := os.DirFS("/")
		file, err := fs.Open("file")
		Expect(err).ToNot(HaveOccurred())
		Expect(file).ToNot(BeNil())

		dir, err := fs.Open("dir")
		Expect(err).ToNot(HaveOccurred())
		Expect(dir).ToNot(BeNil())

		child, err := fs.Open("dir/child")
		Expect(err).ToNot(HaveOccurred())
		Expect(child).ToNot(BeNil())
	})

	It("opened from dir", func() {
		fs := os.DirFS("/dir")
		file, err := fs.Open("../file")
		Expect(err).ToNot(HaveOccurred())
		Expect(file).ToNot(BeNil())

		dir, err := fs.Open("../dir")
		Expect(err).ToNot(HaveOccurred())
		Expect(dir).ToNot(BeNil())

		child, err := fs.Open("child")
		Expect(err).ToNot(HaveOccurred())
		Expect(child).ToNot(BeNil())
	})
})

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

	"github.com/deckhouse/sds-common-lib/fs/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RWContent", func() {
	It("creates empty", func() {
		c := fake.NewRWContent()
		Expect(c.GetBytes()).To(HaveLen(0))
		Expect(c.Size()).To(BeZero())
	})
	It("RWContentFromBytes", func() {
		b := make([]byte, 10)
		for i := range b {
			b[i] = byte(i)
		}
		c := fake.RWContentFromBytes(b)
		Expect(c.GetBytes()).To(BeEquivalentTo(b))
		Expect(c.Size()).To(BeEquivalentTo(len(b)))
	})

	When("hello content", func() {
		s := "Hello"
		var c *fake.RWContent
		JustBeforeEach(func() {
			c = fake.RWContentFromString(s)
		})
		It("RWContentFromString", func() {
			Expect(c.GetString()).To(BeEquivalentTo(s))
			Expect(c.Size()).To(BeEquivalentTo(len(s)))
		})

		It("ReadAt", func() {
			b := make([]byte, 4)
			Expect(c.ReadAt(b, 1)).To(Equal(4))
			Expect(b).To(BeEquivalentTo(s[1:5]))
			Expect(c.Size()).To(BeEquivalentTo(5))
		})

		It("ReadAt over end", func() {
			b := make([]byte, 4)
			Expect(c.ReadAt(b, 4)).To(Equal(1))
			Expect(b[0:1]).To(BeEquivalentTo(s[4:5]))
			Expect(c.Size()).To(BeEquivalentTo(5))
		})

		It("ReadAt pass end", func() {
			b := make([]byte, 4)
			_, err := c.ReadAt(b, 5)
			Expect(err).To(MatchError(io.EOF))
		})

		It("WriteAt from zero", func() {
			newStr := "world"
			b := ([]byte)(newStr)
			Expect(c.WriteAt(b, 0)).To(Equal(len(b)))
			Expect(c.GetString()).To(Equal(newStr))
			Expect(c.Size()).To(BeEquivalentTo(len(b)))
		})

		It("WriteAt from middle", func() {
			newStr := "LL"
			b := ([]byte)(newStr)
			Expect(c.WriteAt(b, 2)).To(Equal(len(b)))
			Expect(c.GetString()).To(Equal("HeLLo"))
			Expect(c.Size()).To(BeEquivalentTo(5))
		})

		It("WriteAt from end", func() {
			newStr := "World"
			b := ([]byte)(newStr)
			Expect(c.WriteAt(b, 5)).To(Equal(len(b)))
			Expect(c.GetString()).To(Equal("HelloWorld"))
			Expect(c.Size()).To(BeEquivalentTo(10))
		})

		It("WriteAt past end", func() {
			newStr := "World"
			b := ([]byte)(newStr)
			Expect(c.WriteAt(b, 6)).To(Equal(len(b)))
			Expect(c.GetString()).To(Equal("Hello\x00World"))
			Expect(c.Size()).To(BeEquivalentTo(11))
		})
	})

})

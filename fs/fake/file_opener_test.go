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
	"errors"
	"fmt"

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/deckhouse/sds-common-lib/fs/mock"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileOpener", func() {
	var builder *fake.Builder
	var ctrl *gomock.Controller
	var sizer *mock.MockFileSizer
	var writer *mock.MockWriter
	var reader *mock.MockReader
	var writerAt *mock.MockWriterAt
	var readerAt *mock.MockReaderAt
	var dirReader *mock.MockDirReader

	JustBeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		sizer = mock.NewMockFileSizer(ctrl)
		sizer.EXPECT().Size().AnyTimes().Return(int64(3))

		reader = mock.NewMockReader(ctrl)
		reader.EXPECT().Read(gomock.Any()).AnyTimes().Return(3, nil)

		writer = mock.NewMockWriter(ctrl)
		writer.EXPECT().Write(gomock.Any()).AnyTimes().Return(3, nil)

		writerAt = mock.NewMockWriterAt(ctrl)
		writerAt.EXPECT().WriteAt(gomock.Any(), gomock.Any()).AnyTimes().Return(3, nil)

		readerAt = mock.NewMockReaderAt(ctrl)
		readerAt.EXPECT().ReadAt(gomock.Any(), gomock.Any()).AnyTimes().Return(3, nil)

		dirReader = mock.NewMockDirReader(ctrl)
		dirReader.EXPECT().ReadDir(gomock.Any()).AnyTimes().Return(make([]fs.DirEntry, 0), nil)

		builder = fake.NewBuilder("/")
		Expect(builder).ToNot(BeNil())
	})
	When("file created and opened", func() {
		var args []any
		var file fs.File
		var createError, openError error
		var fakeOS *fake.OS
		JustBeforeEach(func() {
			fakeOS, createError = builder.WithFile("file", args...).Build()
			if fakeOS != nil {
				file, openError = fakeOS.Open("file")
			} else {
				file = nil
				openError = nil
			}
		})

		itSupports := func(fName string, b bool, fn func() error) {
			name := fmt.Sprintf("does support %s", fName)
			if !b {
				name = fmt.Sprintf("doesn't support %s", fName)
			}

			It(name, func() {
				err := fn()
				if b {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError(errors.ErrUnsupported))
				}
			})
		}

		itSupportsRead := func(b bool) {
			itSupports("Read", b, func() error {
				p := make([]byte, 2)
				_, err := file.Read(p)
				return err
			})
		}

		itSupportsWrite := func(b bool) {
			itSupports("Write", b, func() error {
				p := make([]byte, 2)
				_, err := file.Write(p)
				return err
			})
		}

		itSupportsReadAt := func(b bool) {
			itSupports("ReadAt", b, func() error {
				p := make([]byte, 2)
				_, err := file.ReadAt(p, 0)
				return err
			})
		}

		itSupportsWriteAt := func(b bool) {
			itSupports("WriteAt", b, func() error {
				p := make([]byte, 2)
				_, err := file.WriteAt(p, 0)
				return err
			})
		}

		itSupportsSeek := func(b bool) {
			itSupports("Seek", b, func() error {
				_, err := file.Seek(0, 0)
				return err
			})
		}

		itSupportsReadDir := func(b bool) {
			itSupports("ReadDir", b, func() error {
				_, err := file.ReadDir(1)
				return err
			})
		}

		itSupportsClose := func(b bool) {
			itSupports("Close", b, func() error {
				return file.Close()
			})
		}

		When("no arguments", func() {
			BeforeEach(func() {
				args = make([]any, 0)
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(false)
			itSupportsReadAt(false)
			itSupportsWrite(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("sizer argument", func() {
			BeforeEach(func() {
				args = []any{sizer}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(3))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(false)
			itSupportsReadAt(false)
			itSupportsWrite(false)
			itSupportsWriteAt(false)
			itSupportsSeek(true)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("reader argument", func() {
			BeforeEach(func() {
				args = []any{reader}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(true)
			itSupportsReadAt(false)
			itSupportsWrite(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("reader and NoReader argument", func() {
			BeforeEach(func() {
				args = []any{reader, fake.NoReader}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(false)
			itSupportsReadAt(false)
			itSupportsWrite(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("reader and NoWriter argument", func() {
			BeforeEach(func() {
				args = []any{reader, fake.NoWriter}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(true)
			itSupportsReadAt(false)
			itSupportsWrite(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("writer argument", func() {
			BeforeEach(func() {
				args = []any{writer}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(false)
			itSupportsWrite(true)
			itSupportsReadAt(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("writer with NoWriter argument", func() {
			BeforeEach(func() {
				args = []any{writer, fake.NoWriter}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsReadAt(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})
		When("writerAt argument", func() {
			BeforeEach(func() {
				args = []any{writerAt}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsWriteAt(true)
			itSupportsClose(true)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsReadAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
		})

		When("writerAt and sizer argument", func() {
			BeforeEach(func() {
				args = []any{writerAt, sizer}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(3))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsWriteAt(true)
			itSupportsClose(true)
			itSupportsWrite(true)
			itSupportsSeek(true)

			itSupportsRead(false)
			itSupportsReadAt(false)
			itSupportsReadDir(false)
		})

		When("readerAt and sizer argument", func() {
			BeforeEach(func() {
				args = []any{sizer, readerAt}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(3))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsClose(true)
			itSupportsSeek(true)
			itSupportsRead(true)
			itSupportsReadAt(true)

			itSupportsWriteAt(false)
			itSupportsWrite(false)
			itSupportsReadDir(false)
		})

		When("reader and writer argument", func() {
			BeforeEach(func() {
				args = []any{reader, writer}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsRead(true)
			itSupportsWrite(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})

		When("readerAt and writerAt argument", func() {
			BeforeEach(func() {
				args = []any{readerAt, writerAt}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeZero())
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsReadAt(true)
			itSupportsWriteAt(true)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
			itSupportsReadDir(false)
			itSupportsClose(true)
		})

		When("readerAt, writerAt and sizer argument", func() {
			BeforeEach(func() {
				args = []any{readerAt, writerAt, sizer}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(3))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsReadAt(true)
			itSupportsWriteAt(true)

			itSupportsRead(true)
			itSupportsWrite(true)
			itSupportsSeek(true)

			itSupportsReadDir(false)
			itSupportsClose(true)
		})

		When("readerAt, writerAt, ReadOnly and sizer argument", func() {
			BeforeEach(func() {
				args = []any{readerAt, writerAt, sizer, fake.ReadOnly}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(3))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsReadAt(true)
			itSupportsWriteAt(false)

			itSupportsRead(true)
			itSupportsWrite(false)
			itSupportsSeek(true)

			itSupportsReadDir(false)
			itSupportsClose(true)
		})

		When("dirReader argument", func() {
			BeforeEach(func() {
				args = []any{dirReader}
			})
			JustBeforeEach(func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})
			It("has correct stat", func() {
				fi, err := file.Stat()
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.Size()).To(BeEquivalentTo(0))
				Expect(fi.IsDir()).To(BeFalse())
				Expect(fi.Name()).To(BeEquivalentTo("file"))
			})
			itSupportsReadDir(true)
			itSupportsClose(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
		})
		When("dirReader and reader argument", func() {
			BeforeEach(func() {
				args = []any{dirReader, reader}
			})
			It("fails to create file", func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})

			itSupportsReadDir(true)
			itSupportsClose(true)
			itSupportsRead(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsWrite(false)
			itSupportsSeek(false)
		})

		When("dirReader and reader argument and modeDir", func() {
			BeforeEach(func() {
				args = []any{dirReader, reader, fs.ModeDir}
			})
			It("creates file", func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})

			itSupportsReadDir(true)
			itSupportsClose(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
		})

		When("dirReader and readerAt argument and modeDir", func() {
			BeforeEach(func() {
				args = []any{dirReader, readerAt, fs.ModeDir}
			})
			It("creates file", func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})

			itSupportsReadDir(true)
			itSupportsClose(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
		})

		When("dirReader and writerAt argument and modeDir", func() {
			BeforeEach(func() {
				args = []any{dirReader, writerAt, fs.ModeDir}
			})
			It("creates file", func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})

			itSupportsReadDir(true)
			itSupportsClose(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
		})

		When("dirReader and writer argument and modeDir", func() {
			BeforeEach(func() {
				args = []any{dirReader, writer, fs.ModeDir}
			})
			It("creates file", func() {
				Expect(createError).ToNot(HaveOccurred())
				Expect(openError).ToNot(HaveOccurred())
			})

			itSupportsReadDir(true)
			itSupportsClose(true)

			itSupportsReadAt(false)
			itSupportsWriteAt(false)

			itSupportsRead(false)
			itSupportsWrite(false)
			itSupportsSeek(false)
		})
	})

})

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

	"github.com/deckhouse/sds-common-lib/fs"
	"github.com/deckhouse/sds-common-lib/fs/fake"
	"github.com/deckhouse/sds-common-lib/fs/mock"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("closer", func() {
	When("MockReader created", func() {
		var m *mock.MockReader
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockReader(ctrl)
		})

		When("closer created by reader", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().Read(p).Return(10, nil)
				Expect(closer.Read(p)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Write(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.Read(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes Read error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().Read(p).Return(10, e)
				_, err := closer.Read(p)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockWriter created", func() {
		var m *mock.MockWriter
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockWriter(ctrl)
		})

		When("closer created", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Write", func() {
				p := make([]byte, 12)
				m.EXPECT().Write(p).Return(10, nil)
				Expect(closer.Write(p)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Read(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails", func() {
					p := make([]byte, 12)
					_, err := closer.Write(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().Write(p).Return(10, e)
				_, err := closer.Write(p)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockReaderWriter created", func() {
		var m *mock.MockReadWriter
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockReadWriter(ctrl)
		})

		When("closer created", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Write", func() {
				p := make([]byte, 12)
				m.EXPECT().Write(p).Return(10, nil)
				Expect(closer.Write(p)).To(BeEquivalentTo(10))
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().Read(p).Return(10, nil)
				Expect(closer.Read(p)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to write", func() {
					p := make([]byte, 12)
					_, err := closer.Write(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.Read(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().Write(p).Return(10, e)
				_, err := closer.Write(p)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockReadWriteCloser created", func() {
		var m *mock.MockReadWriteCloser
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockReadWriteCloser(ctrl)
		})

		When("closer created", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Write", func() {
				p := make([]byte, 12)
				m.EXPECT().Write(p).Return(10, nil)
				Expect(closer.Write(p)).To(BeEquivalentTo(10))
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().Read(p).Return(10, nil)
				Expect(closer.Read(p)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					m.EXPECT().Close()
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to write", func() {
					p := make([]byte, 12)
					_, err := closer.Write(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.Read(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().Write(p).Return(10, e)
				_, err := closer.Write(p)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockReadSeeker created", func() {
		var m *mock.MockReadSeeker
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockReadSeeker(ctrl)
		})

		When("closer created by reader", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().Read(p).Return(10, nil)
				Expect(closer.Read(p)).To(BeEquivalentTo(10))
			})

			It("passes Seek", func() {
				m.EXPECT().Seek(int64(1), 2).Return(int64(3), nil)
				Expect(closer.Seek(1, 2)).To(BeEquivalentTo(int64(3)))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Write(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.Read(p)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes Read error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().Read(p).Return(10, e)
				_, err := closer.Read(p)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockReaderAt created", func() {
		var m *mock.MockReaderAt
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockReaderAt(ctrl)
		})

		When("closer created by reader", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().ReadAt(p, int64(1)).Return(10, nil)
				Expect(closer.ReadAt(p, 1)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.Read(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Write(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.ReadAt(p, 1)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes ReadAt error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().ReadAt(p, int64(1)).Return(10, e)
				_, err := closer.ReadAt(p, 1)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockWriterAt created", func() {
		var m *mock.MockWriterAt
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockWriterAt(ctrl)
		})

		When("closer created by reader", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Read", func() {
				p := make([]byte, 12)
				m.EXPECT().WriteAt(p, int64(1)).Return(10, nil)
				Expect(closer.WriteAt(p, 1)).To(BeEquivalentTo(10))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.Read(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadDir(1)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Write(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					p := make([]byte, 12)
					_, err := closer.WriteAt(p, 1)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes ReadAt error", func() {
				p := make([]byte, 12)
				e := errors.New("my error")
				m.EXPECT().WriteAt(p, int64(1)).Return(10, e)
				_, err := closer.WriteAt(p, 1)
				Expect(err).To(MatchError(e))
			})

		})
	})

	When("MockDirReader created", func() {
		var m *mock.MockDirReader
		JustBeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = mock.NewMockDirReader(ctrl)
		})

		When("closer created by reader", func() {
			var closer *fake.Closer
			var err error
			JustBeforeEach(func() {
				closer, err = fake.NewCloser(m)
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes Read", func() {
				p := make([]fs.DirEntry, 12)
				m.EXPECT().ReadDir(1).Return(p, nil)
				Expect(closer.ReadDir(1)).To(BeEquivalentTo(p))
			})

			It("does not support the rest", func() {
				p := make([]byte, 12)
				_, err := closer.Read(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))
				_, err = closer.Write(p)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.Seek(0, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))

				_, err = closer.ReadAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
				_, err = closer.WriteAt(p, 0)
				Expect(err).To(MatchError(errors.ErrUnsupported))
			})

			When("closed", func() {
				JustBeforeEach(func() {
					Expect(closer.Close()).ToNot(HaveOccurred())
				})
				It("fails to close second time", func() {
					Expect(closer.Close()).To(MatchError(fs.ErrClosed))
				})
				It("fails to read", func() {
					_, err := closer.ReadDir(1)
					Expect(err).To(MatchError(fs.ErrClosed))
				})
			})

			It("passes ReadAt error", func() {
				p := make([]fs.DirEntry, 12)
				e := errors.New("my error")
				m.EXPECT().ReadDir(1).Return(p, e)
				_, err := closer.ReadDir(1)
				Expect(err).To(MatchError(e))
			})

		})
	})
})

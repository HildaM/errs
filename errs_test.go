package errs_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/HildaM/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestErrs(t *testing.T) {
	var err *errs.Error
	str := err.Error()
	assert.Contains(t, str, "success")

	e := errs.New(111, "inner fail")
	assert.NotNil(t, e)

	assert.Equal(t, 111, errs.Code(e))
	assert.Equal(t, "inner fail", errs.Msg(e))

	err, ok := e.(*errs.Error)
	assert.Equal(t, true, ok)
	assert.NotNil(t, err)
	assert.Equal(t, errs.ErrorTypeBusiness, err.Type)

	str = err.Error()
	assert.Contains(t, str, "business")

	e = errs.NewFrameError(111, "inner fail")
	assert.NotNil(t, e)

	assert.Equal(t, 111, errs.Code(e))
	assert.Equal(t, "inner fail", errs.Msg(e))

	err, ok = e.(*errs.Error)
	assert.Equal(t, true, ok)
	assert.NotNil(t, err)
	assert.Equal(t, errs.ErrorTypeFramework, err.Type)

	str = err.Error()
	assert.Contains(t, str, "framework")

	assert.Equal(t, 0, errs.Code(nil))
	assert.Equal(t, "success", errs.Msg(nil))

	assert.Equal(t, 0, errs.Code((*errs.Error)(nil)))
	assert.Equal(t, "success", errs.Msg((*errs.Error)(nil)))

	e = errors.New("unknown error")
	assert.Equal(t, errs.RetUnknown, errs.Code(e))
	assert.Equal(t, "unknown error", errs.Msg(e))

	err.Type = errs.ErrorTypeCalleeFramework
	assert.Contains(t, err.Error(), "type:callee framework")
}

func TestNonEmptyStringOnEmptyMsg(t *testing.T) {
	e := errs.New(errs.RetServerSystemErr, "")
	require.Contains(t, e.Error(), "code:")
	require.Contains(t, e.Error(), "type:")
}

func TestErrsFormat(t *testing.T) {
	err := errs.New(10000, "test error")

	s := fmt.Sprintf("%s", err)
	assert.Equal(t, "type:business, code:10000, msg:test error", s)

	s = fmt.Sprintf("%q", err)
	assert.Equal(t, `"type:business, code:10000, msg:test error"`, s)

	s = fmt.Sprintf("%v", err)
	assert.Equal(t, "type:business, code:10000, msg:test error", s)

	s = fmt.Sprintf("%d", err)
	assert.Equal(t, "%!d(errs.Error=type:business, code:10000, msg:test error)", s)
}

func TestNewFrameError(t *testing.T) {
	ok := true
	errs.SetTraceable(ok)
	e := errs.NewFrameError(111, "inner fail")
	assert.NotNil(t, e)
	e = errs.NewCalleeFrameError(111, "callee frame error")
	assert.NotNil(t, e)
}

func TestWrapFrameError(t *testing.T) {
	ok := true
	errs.SetTraceable(ok)
	e := errs.WrapFrameError(errs.New(123, "inner fail"), 456, "wrap frame error")
	assert.NotNil(t, e)
	e = errs.WrapFrameError(nil, 456, "wrap frame error")
	assert.Nil(t, e)
}

func TestTraceError(t *testing.T) {

	errs.SetTraceable(true)

	err := parent()
	assert.NotNil(t, err)

	s := fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))

	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:111, msg:inner fail", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	_, _, _ = br.ReadLine()
	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	_, _, _ = br.ReadLine()
	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func TestTraceErrorSetStackSkip(t *testing.T) {
	errs.SetTraceable(true)
	errs.SetStackSkip(4)

	err := func() error {
		return func() error {
			return newMyErr(11, "TestTraceErrorSetStackSkip error")
		}()
	}()
	assert.NotNil(t, err)

	s := fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))

	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:11, msg:TestTraceErrorSetStackSkip error", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	t.Log(string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	_, _, _ = br.ReadLine()
	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	_, _, _ = br.ReadLine()
	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func newMyErr(code int, msg string) error {
	return errs.New(code, msg)
}

// TestSetTraceableWithContent SetTraceableWithContent interface test case,
// filter and print stack information according to Content.
func TestSetTraceableWithContent(t *testing.T) {
	errs.SetTraceableWithContent("child")

	err := parent()
	assert.NotNil(t, err)

	s := fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))
	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:111, msg:inner fail", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func TestErrorChain(t *testing.T) {
	var e error = errs.Wrap(os.ErrDeadlineExceeded, errs.RetClientTimeout, "just wrap")
	require.Contains(t, errs.Msg(e), os.ErrDeadlineExceeded.Error())
	e = fmt.Errorf("%w", e)
	require.Equal(t, errs.RetClientTimeout, errs.Code(e))
	require.True(t, errors.Is(e, os.ErrDeadlineExceeded))
	require.Contains(t, e.Error(), os.ErrDeadlineExceeded.Error())
}

func TestWrap(t *testing.T) {
	err := parent()
	assert.NotNil(t, err)

	err = errs.Wrap(err, 222, "wrap err")
	assert.NotNil(t, err)

	s := fmt.Sprintf("%v", err)
	assert.Contains(t, s, "type:business, code:222, msg:wrap err")
	s = fmt.Sprintf("%s", err)
	assert.Contains(t, s, "type:business, code:222, msg:wrap err")

	s = fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))
	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:222, msg:wrap err", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, "Cause by type:business, code:111, msg:inner fail", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func TestWrapf(t *testing.T) {
	err := parent()
	assert.NotNil(t, err)

	err = errs.Wrapf(err, 222, "wrap %v", "err")
	assert.NotNil(t, err)

	s := fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))
	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:222, msg:wrap err", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, "Cause by type:business, code:111, msg:inner fail", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func TestWrapSetTraceable(t *testing.T) {
	// reset
	errs.SetStackSkip(3)
	errs.SetTraceableWithContent("")

	err := parent()
	assert.NotNil(t, err)

	err = errs.Wrap(err, 222, "wrap err")
	assert.NotNil(t, err)

	s := fmt.Sprintf("%+v", err)
	br := bufio.NewReader(strings.NewReader(s))
	line, isPrefix, err := br.ReadLine()
	assert.Equal(t, "type:business, code:222, msg:wrap err", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, "Cause by type:business, code:111, msg:inner fail", string(line))
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)

	line, isPrefix, err = br.ReadLine()
	assert.Equal(t, isPrefix, false)
	assert.Nil(t, err)
}

func TestIsTimeout(t *testing.T) {
	require.True(t, (&errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetClientTimeout,
	}).IsTimeout(errs.ErrorTypeFramework))
	require.True(t, (&errs.Error{
		Type: errs.ErrorTypeCalleeFramework,
		Code: errs.RetClientTimeout,
	}).IsTimeout(errs.ErrorTypeCalleeFramework))
	require.False(t, (&errs.Error{
		Type: errs.ErrorTypeBusiness,
		Code: errs.RetClientTimeout,
	}).IsTimeout(errs.ErrorTypeFramework))
	require.True(t, (&errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetClientFullLinkTimeout,
	}).IsTimeout(errs.ErrorTypeFramework))
	require.True(t, (&errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetServerTimeout,
	}).IsTimeout(errs.ErrorTypeFramework))
	require.True(t, (&errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetServerFullLinkTimeout,
	}).IsTimeout(errs.ErrorTypeFramework))
	require.False(t, (&errs.Error{
		Type: errs.ErrorTypeFramework,
		Code: errs.RetServerNoService,
	}).IsTimeout(errs.ErrorTypeFramework))
}

func TestErrorFormatPrint(t *testing.T) {
	errs.SetTraceable(false)
	defer errs.SetTraceable(true)
	err := errs.New(errs.ErrorTypeFramework, "")
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%+v", err)
	require.Equal(t, "type:business, code:1, msg:", buf.String())
}

func TestNestErrors(t *testing.T) {
	errs.SetTraceable(false)
	defer errs.SetTraceable(true)
	const (
		code = 101
		msg  = "test error"
	)
	require.Equal(t, code, errs.Code(&testError{Err: errs.New(code, msg)}))
	require.Equal(t, msg, errs.Msg(&testError{Err: errs.New(code, msg)}))
}

func TestNilErrorUnwrap(t *testing.T) {
	var err *errs.Error
	// Check nil error should not result in panic.
	require.False(t, errors.Is(err, errors.New("some error")))
}

type testError struct {
	Err error
}

func (te *testError) Error() string {
	return te.Err.Error()
}

func (te *testError) Unwrap() error {
	return te.Err
}

//go:noinline
func parent() error {
	if err := child(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func child() error {
	if err := grandson(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func grandson() error {
	return errs.Newf(111, "%s", "inner fail")
}

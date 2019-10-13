package idmanager

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDManager_ReserveNextID(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		nextID, free, err := reserveNextID()
		require.NoError(t, err)
		require.NotNil(t, free)
		v, ok := values[*nextID]
		require.True(t, ok)
		require.Nil(t, v)
		require.Equal(t, *nextID, uint16(1))
		require.Equal(t, *nextID, lstID)

		nextID, free, err = reserveNextID()
		require.NoError(t, err)
		require.NotNil(t, free)
		v, ok = values[*nextID]
		require.True(t, ok)
		require.Nil(t, v)
		require.Equal(t, *nextID, uint16(2))
		require.Equal(t, *nextID, lstID)
	})

	t.Run("call on full manager", func(t *testing.T) {
		m := newIDManager()
		for i := uint16(0); i < math.MaxUint16; i++ {
			values[i] = nil
		}
		values[math.MaxUint16] = nil

		_, _, err := reserveNextID()
		require.Error(t, err)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		valsToReserve := 10000

		errs := make(chan error)
		for i := 0; i < valsToReserve; i++ {
			go func() {
				_, _, err := reserveNextID()
				errs <- err
			}()
		}

		for i := 0; i < valsToReserve; i++ {
			require.NoError(t, <-errs)
		}
		close(errs)

		require.Equal(t, lstID, uint16(valsToReserve))
		for i := uint16(1); i < uint16(valsToReserve); i++ {
			v, ok := values[i]
			require.True(t, ok)
			require.Nil(t, v)
		}
	})
}

func TestIDManager_Pop(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		v := "value"

		values[1] = v

		gotV, err := pop(1)
		require.NoError(t, err)
		require.NotNil(t, gotV)
		require.Equal(t, gotV, v)

		_, ok := values[1]
		require.False(t, ok)
	})

	t.Run("no value", func(t *testing.T) {
		m := newIDManager()

		_, err := pop(1)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "no value"))
	})

	t.Run("value not set", func(t *testing.T) {
		m := newIDManager()

		values[1] = nil

		_, err := pop(1)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "is not set"))
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		values[1] = "value"

		concurrency := 1000
		errs := make(chan error, concurrency)
		for i := uint16(0); i < uint16(concurrency); i++ {
			go func() {
				_, err := pop(1)
				errs <- err
			}()
		}

		errsCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errs
			if err != nil {
				errsCount++
			}
		}
		close(errs)
		require.Equal(t, errsCount, concurrency-1)

		_, ok := values[1]
		require.False(t, ok)
	})
}

func TestIDManager_Add(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		id := uint16(1)
		v := "value"

		free, err := add(id, v)
		require.Nil(t, err)
		require.NotNil(t, free)

		gotV, ok := values[id]
		require.True(t, ok)
		require.Equal(t, gotV, v)

		v2 := "value2"

		free, err = add(id, v2)
		require.Equal(t, err, errValueAlreadyExists)
		require.Nil(t, free)

		gotV, ok = values[id]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		id := uint16(1)

		concurrency := 1000

		addV := make(chan int)
		errs := make(chan error)
		for i := 0; i < concurrency; i++ {
			go func(v int) {
				_, err := add(id, v)
				errs <- err
				if err == nil {
					addV <- v
				}
			}(i)
		}

		errsCount := 0
		for i := 0; i < concurrency; i++ {
			if err := <-errs; err != nil {
				errsCount++
			}
		}
		close(errs)

		v := <-addV
		close(addV)

		require.Equal(t, concurrency-1, errsCount)

		gotV, ok := values[id]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})
}

func TestIDManager_Set(t *testing.T) {
	t.Run("simple call", func(t *testing.T) {
		m := newIDManager()

		nextID, _, err := reserveNextID()
		require.NoError(t, err)

		v := "value"

		err = set(*nextID, v)
		require.NoError(t, err)
		gotV, ok := values[*nextID]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})

	t.Run("id is not reserved", func(t *testing.T) {
		m := newIDManager()

		err := set(1, "value")
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "not reserved"))

		_, ok := values[1]
		require.False(t, ok)
	})

	t.Run("value already exists", func(t *testing.T) {
		m := newIDManager()

		v := "value"

		values[1] = v

		err := set(1, "value2")
		require.Error(t, err)
		gotV, ok := values[1]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})

	t.Run("concurrent run", func(t *testing.T) {
		m := newIDManager()

		concurrency := 1000

		nextIDPtr, _, err := reserveNextID()
		require.NoError(t, err)

		nextID := *nextIDPtr

		errs := make(chan error)
		setV := make(chan int)
		for i := 0; i < concurrency; i++ {
			go func(v int) {
				err := set(nextID, v)
				errs <- err
				if err == nil {
					setV <- v
				}
			}(i)
		}

		errsCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-errs
			if err != nil {
				errsCount++
			}
		}
		close(errs)

		v := <-setV
		close(setV)

		require.Equal(t, concurrency-1, errsCount)

		gotV, ok := values[nextID]
		require.True(t, ok)
		require.Equal(t, gotV, v)
	})
}

func TestIDManager_Get(t *testing.T) {
	prepManagerWithVal := func(v interface{}) (*idManager, uint16) {
		m := newIDManager()

		nextID, _, err := reserveNextID()
		require.NoError(t, err)

		err = set(*nextID, v)
		require.NoError(t, err)

		return m, *nextID
	}

	t.Run("simple call", func(t *testing.T) {
		v := "value"

		m, id := prepManagerWithVal(v)

		gotV, ok := get(id)
		require.True(t, ok)
		require.Equal(t, gotV, v)

		_, ok = get(100)
		require.False(t, ok)

		values[2] = nil
		gotV, ok = get(2)
		require.False(t, ok)
		require.Nil(t, gotV)
	})

	t.Run("concurrent run", func(t *testing.T) {
		v := "value"

		m, id := prepManagerWithVal(v)

		concurrency := 1000
		type getRes struct {
			v  interface{}
			ok bool
		}
		res := make(chan getRes)
		for i := 0; i < concurrency; i++ {
			go func() {
				val, ok := get(id)
				res <- getRes{
					v:  val,
					ok: ok,
				}
			}()
		}

		for i := 0; i < concurrency; i++ {
			r := <-res
			require.True(t, r.ok)
			require.Equal(t, r.v, v)
		}
		close(res)
	})
}

func TestIDManager_DoRange(t *testing.T) {
	m := newIDManager()

	valsCount := 5

	vals := make([]int, 0, valsCount)
	for i := 0; i < valsCount; i++ {
		vals = append(vals, i)
	}

	for i, v := range vals {
		_, err := add(uint16(i), v)
		require.NoError(t, err)
	}

	// run full range
	gotVals := make([]int, 0, valsCount)
	doRange(func(_ uint16, v interface{}) bool {
		val, ok := v.(int)
		require.True(t, ok)

		gotVals = append(gotVals, val)

		return true
	})
	sort.Ints(gotVals)
	require.Equal(t, gotVals, vals)

	// run part range
	var gotVal int
	gotValsCount := 0
	doRange(func(_ uint16, v interface{}) bool {
		if gotValsCount == 1 {
			return false
		}

		val, ok := v.(int)
		require.True(t, ok)

		gotVal = val

		gotValsCount++

		return true
	})

	found := false
	for _, v := range vals {
		if v == gotVal {
			found = true
		}
	}
	require.True(t, found)
}

func TestIDManager_ConstructFreeFunc(t *testing.T) {
	m := newIDManager()

	id := uint16(1)
	v := "value"

	free, err := add(id, v)
	require.NoError(t, err)
	require.NotNil(t, free)

	free()

	gotV, ok := values[id]
	require.False(t, ok)
	require.Nil(t, gotV)
}
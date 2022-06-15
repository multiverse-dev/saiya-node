package result

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/pkg/core/interop"
	"github.com/nspcc-dev/neo-go/pkg/core/interop/iterator"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/storage"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
)

// Invoke represents a code invocation result and is used by several RPC calls
// that invoke functions, scripts and generic bytecode.
type Invoke struct {
	State          string
	GasConsumed    int64
	Script         []byte
	Stack          []stackitem.Item
	FaultException string
	Notifications  []state.NotificationEvent
	Transaction    *transaction.Transaction
	Diagnostics    *InvokeDiag
	Session        uuid.UUID
	finalize       func()
	onNewSession   OnNewSession
}

type OnNewSession func(sessionID string, iterators []ServerIterator, finalize func())

// InvokeDiag is an additional diagnostic data for invocation.
type InvokeDiag struct {
	Changes     []storage.Operation  `json:"storagechanges"`
	Invocations []*vm.InvocationTree `json:"invokedcontracts"`
}

// NewInvoke returns a new Invoke structure with the given fields set.
func NewInvoke(ic *interop.Context, script []byte, faultException string, registerSession OnNewSession) *Invoke {
	var diag *InvokeDiag
	tree := ic.VM.GetInvocationTree()
	if tree != nil {
		diag = &InvokeDiag{
			Invocations: tree.Calls,
			Changes:     storage.BatchToOperations(ic.DAO.GetBatch()),
		}
	}
	notifications := ic.Notifications
	if notifications == nil {
		notifications = make([]state.NotificationEvent, 0)
	}
	return &Invoke{
		State:          ic.VM.State().String(),
		GasConsumed:    ic.VM.GasConsumed(),
		Script:         script,
		Stack:          ic.VM.Estack().ToArray(),
		FaultException: faultException,
		Notifications:  notifications,
		Diagnostics:    diag,
		finalize:       ic.Finalize,
		onNewSession:   registerSession,
	}
}

type invokeAux struct {
	State          string                    `json:"state"`
	GasConsumed    int64                     `json:"gasconsumed,string"`
	Script         []byte                    `json:"script"`
	Stack          json.RawMessage           `json:"stack"`
	FaultException *string                   `json:"exception"`
	Notifications  []state.NotificationEvent `json:"notifications"`
	Transaction    []byte                    `json:"tx,omitempty"`
	Diagnostics    *InvokeDiag               `json:"diagnostics,omitempty"`
	Session        string                    `json:"session,omitempty"`
}

// iteratorInterfaceName is a string used to mark Iterator inside the InteropInterface.
const iteratorInterfaceName = "IIterator"

type iteratorAux struct {
	Type      string `json:"type"`
	Interface string `json:"interface"`
	ID        string `json:"id"`
}

// Iterator represents VM iterator identifier.
type Iterator struct {
	ID uuid.UUID
}

// ServerIterator represents Iterator on the server side.
type ServerIterator struct {
	ID   string
	Item stackitem.Item
}

// Finalize releases resources occupied by Iterators created at the script invocation.
// This method will be called automatically on Invoke marshalling or by the Server's
// sessions handler.
func (r *Invoke) Finalize() {
	if r.finalize != nil {
		r.finalize()
	}
}

// MarshalJSON implements the json.Marshaler.
func (r Invoke) MarshalJSON() ([]byte, error) {
	var (
		st              json.RawMessage
		err             error
		faultSep        string
		arr             = make([]json.RawMessage, len(r.Stack))
		sessionsEnabled = r.onNewSession != nil
		sessionID       string
		iterators       []ServerIterator
	)
	if len(r.FaultException) != 0 {
		faultSep = " / "
	}
	for i := range arr {
		var data []byte
		if (r.Stack[i].Type() == stackitem.InteropT) && iterator.IsIterator(r.Stack[i]) {
			iteratorID := uuid.NewString()
			data, err = json.Marshal(iteratorAux{
				Type:      stackitem.InteropT.String(),
				Interface: iteratorInterfaceName,
				ID:        iteratorID,
			})
			if err != nil {
				r.FaultException += fmt.Sprintf("%sjson error: failed to marshal iterator: %v", faultSep, err)
				break
			}
			if sessionsEnabled {
				iterators = append(iterators, ServerIterator{
					ID:   iteratorID,
					Item: r.Stack[i],
				})
			}
		} else {
			data, err = stackitem.ToJSONWithTypes(r.Stack[i])
			if err != nil {
				r.FaultException += fmt.Sprintf("%sjson error: %v", faultSep, err)
				break
			}
		}
		arr[i] = data
	}

	if sessionsEnabled && len(iterators) != 0 {
		sessionID = uuid.NewString()
		r.onNewSession(sessionID, iterators, r.Finalize)
	} else {
		defer r.Finalize()
	}

	if err == nil {
		st, err = json.Marshal(arr)
		if err != nil {
			return nil, err
		}
	}
	var txbytes []byte
	if r.Transaction != nil {
		txbytes = r.Transaction.Bytes()
	}
	aux := &invokeAux{
		GasConsumed:   r.GasConsumed,
		Script:        r.Script,
		State:         r.State,
		Stack:         st,
		Notifications: r.Notifications,
		Transaction:   txbytes,
		Diagnostics:   r.Diagnostics,
		Session:       sessionID,
	}
	if len(r.FaultException) != 0 {
		aux.FaultException = &r.FaultException
	}
	return json.Marshal(aux)
}

// UnmarshalJSON implements the json.Unmarshaler.
func (r *Invoke) UnmarshalJSON(data []byte) error {
	var err error
	aux := new(invokeAux)
	if err = json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.Session) != 0 {
		r.Session, err = uuid.Parse(aux.Session)
		if err != nil {
			return fmt.Errorf("failed to parse session ID: %w", err)
		}
	}
	var arr []json.RawMessage
	if err = json.Unmarshal(aux.Stack, &arr); err == nil {
		st := make([]stackitem.Item, len(arr))
		for i := range arr {
			st[i], err = stackitem.FromJSONWithTypes(arr[i])
			if err != nil {
				break
			}
			if st[i].Type() == stackitem.InteropT {
				iteratorAux := new(iteratorAux)
				if json.Unmarshal(arr[i], iteratorAux) == nil {
					if iteratorAux.Interface != iteratorInterfaceName {
						err = fmt.Errorf("unknown InteropInterface: %s", iteratorAux.Interface)
					}
					iID, err := uuid.Parse(iteratorAux.ID) // iteratorAux.ID is always non-empty, see https://github.com/neo-project/neo-modules/pull/715#discussion_r897635424.
					if err != nil {
						break
					}
					// it's impossible to restore initial iterator type; also iterator is almost
					// useless outside the VM, thus let's replace it with a special structure.
					st[i] = stackitem.NewInterop(Iterator{
						ID: iID,
					})
				}
			}
		}
		if err == nil {
			r.Stack = st
		}
	}
	var tx *transaction.Transaction
	if len(aux.Transaction) != 0 {
		tx, err = transaction.NewTransactionFromBytes(aux.Transaction)
		if err != nil {
			return err
		}
	}
	r.GasConsumed = aux.GasConsumed
	r.Script = aux.Script
	r.State = aux.State
	if aux.FaultException != nil {
		r.FaultException = *aux.FaultException
	}
	r.Notifications = aux.Notifications
	r.Transaction = tx
	r.Diagnostics = aux.Diagnostics
	return nil
}

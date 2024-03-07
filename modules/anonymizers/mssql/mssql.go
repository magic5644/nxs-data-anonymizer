package mssql_anonymize

import (
	"context"
	"io"
	"strings"

	"github.com/nixys/nxs-data-anonymizer/modules/filters/relfilter"

	fsm "github.com/nixys/nxs-go-fsm"
)

type userCtx struct {
	filter *relfilter.Filter
	column userColumnCtx
}

type userColumnCtx struct {
	name       string
	columnType relfilter.ColumnType
	isSkip     bool
}

var typeKeys = map[string]relfilter.ColumnType{

	// Special
	"uniqueidentifier": relfilter.ColumnTypeNone,

	// Strings
	"char":           relfilter.ColumnTypeString,
	"nchar":          relfilter.ColumnTypeString,
	"varchar":        relfilter.ColumnTypeString,
	"nvarchar":       relfilter.ColumnTypeString,
	"text":           relfilter.ColumnTypeString,
	"ntext":          relfilter.ColumnTypeString,
	"date":           relfilter.ColumnTypeString,
	"datetime":       relfilter.ColumnTypeString,
	"datetime2":      relfilter.ColumnTypeString,
	"datetimeoffset": relfilter.ColumnTypeString,
	"smalldatetime":  relfilter.ColumnTypeString,
	"time":           relfilter.ColumnTypeString,
	"sql_variant":    relfilter.ColumnTypeString,
	"xml":            relfilter.ColumnTypeString,
	"json":           relfilter.ColumnTypeString,

	// Numeric
	"bit":       relfilter.ColumnTypeNum,
	"tinyint":   relfilter.ColumnTypeNum,
	"smallint":  relfilter.ColumnTypeNum,
	"int":       relfilter.ColumnTypeNum,
	"bigint":    relfilter.ColumnTypeNum,
	"decimal":   relfilter.ColumnTypeNum,
	"numeric":   relfilter.ColumnTypeNum,
	"money":     relfilter.ColumnTypeNum,
	"smalmoney": relfilter.ColumnTypeNum,

	// Binary
	"binary":    relfilter.ColumnTypeBinary,
	"varbinary": relfilter.ColumnTypeBinary,
	"image":     relfilter.ColumnTypeBinary,
}

func userCtxInit(rules relfilter.Rules) *userCtx {
	return &userCtx{
		filter: relfilter.Init(rules),
	}
}

func Init(ctx context.Context, r io.Reader, rules relfilter.Rules) io.Reader {

	return fsm.Init(
		r,
		fsm.Description{
			Ctx:       ctx,
			UserCtx:   userCtxInit(rules),
			InitState: stateCreateSearch,
			States: map[fsm.StateName]fsm.State{

				stateCreateSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateCreateTableSearch,
							Switch: fsm.Switch{
								Trigger: []byte("CREATE"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateCreateTableSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateCreateTableNameSearch,
							Switch: fsm.Switch{
								Trigger: []byte("TABLE"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateCreateTableNameSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateCreateTableName,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: nil,
						},
					},
				},
				stateCreateTableName: {
					NextStates: []fsm.NextState{
						{
							Name: stateFieldsDescriptionSearch,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: dhCreateTableName,
						},
					},
				},
				stateFieldsDescriptionSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateFieldsDescriptionBlock,
							Switch: fsm.Switch{
								Trigger: []byte("("),
							},
							DataHandler: nil,
						},
					},
				},
				stateFieldsDescriptionBlock: {
					NextStates: []fsm.NextState{
						{
							// Skip table keys description
							Name: stateFieldDescriptionTailSkip,
							Switch: fsm.Switch{
								Trigger: []byte("KEY"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							// Skip table keys description
							Name: stateFieldDescriptionTailSkip,
							Switch: fsm.Switch{
								Trigger: []byte("PRIMARY"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							// Skip table keys description
							Name: stateFieldDescriptionTailSkip,
							Switch: fsm.Switch{
								Trigger: []byte("UNIQUE"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							// Skip table keys description
							Name: stateFieldDescriptionTailSkip,
							Switch: fsm.Switch{
								Trigger: []byte("CONSTRAINT"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							// Skip table keys description
							Name: stateFieldDescriptionTailSkip,
							Switch: fsm.Switch{
								Trigger: []byte("FOREIGN"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							Name: stateFieldsDescriptionName,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: nil,
						},
					},
				},
				stateFieldDescriptionTailSkip: {
					NextStates: []fsm.NextState{
						{
							Name: stateFieldsDescriptionBlock,
							Switch: fsm.Switch{
								Trigger: []byte(","),
								Delimiters: fsm.Delimiters{
									R: []byte{'\n'},
								},
							},
							DataHandler: nil,
						},
						{
							Name: statefFieldsDescriptionBlockEnd,
							Switch: fsm.Switch{
								Trigger: []byte(")"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateFieldsDescriptionName: {
					NextStates: []fsm.NextState{
						{
							Name: stateFieldsDescriptionNameTail,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: dhCreateTableFieldName,
						},
					},
				},
				stateFieldsDescriptionNameTail: {
					NextStates: func() []fsm.NextState {

						var nss []fsm.NextState

						for t := range typeKeys {
							for i := 0; i < 2; i++ {

								s := t
								if i == 1 {
									s = strings.ToUpper(t)
								}

								nss = append(nss, fsm.NextState{
									Name: stateFieldsDescriptionNameTail,
									Switch: fsm.Switch{
										Trigger: []byte(s),
										Delimiters: fsm.Delimiters{
											L: []byte{' '},
											R: []byte{' ', '(', ',', '\n'},
										},
									},
									DataHandler: dhCreateTableColumnTypeAdd,
								})
							}
						}

						// Additional states
						nss = append(nss, fsm.NextState{
							Name: stateFieldsDescriptionBlock,
							Switch: fsm.Switch{
								Trigger: []byte(","),
								Delimiters: fsm.Delimiters{
									R: []byte{'\n'},
								},
							},
							DataHandler: dhCreateTableColumnAdd,
						})
						nss = append(nss, fsm.NextState{
							Name: statefFieldsDescriptionBlockEnd,
							Switch: fsm.Switch{
								Trigger: []byte(")"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
								},
							},
							DataHandler: dhCreateTableColumnAdd,
						})

						return nss
					}(),
				},
				statefFieldsDescriptionBlockEnd: {
					NextStates: []fsm.NextState{
						{
							Name: stateSomeIntermediateState,
							Switch: fsm.Switch{
								Trigger: []byte(";"),
								Delimiters: fsm.Delimiters{
									R: []byte{'\n'},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateSomeIntermediateState: {
					NextStates: []fsm.NextState{
						{
							Name: stateCreateTableSearch,
							Switch: fsm.Switch{
								Trigger: []byte("CREATE"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
						{
							Name: stateInsertInto,
							Switch: fsm.Switch{
								Trigger: []byte("INSERT"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
					},
				},

				stateInsertInto: {
					NextStates: []fsm.NextState{
						{
							Name: stateInsertIntoTableNameSearch,
							Switch: fsm.Switch{
								Trigger: []byte("INTO"),
								Delimiters: fsm.Delimiters{
									L: []byte{' '},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateInsertIntoTableNameSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateInsertIntoTableName,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: nil,
						},
					},
				},
				stateInsertIntoTableName: {
					NextStates: []fsm.NextState{
						{
							Name: stateValuesSearchKeyword,
							Switch: fsm.Switch{
								Trigger: []byte("`"),
							},
							DataHandler: dhInsertIntoTableName,
						},
					},
				},
				stateValuesSearchKeyword: {
					NextStates: []fsm.NextState{
						{
							Name: stateValuesSearch,
							Switch: fsm.Switch{
								Trigger: []byte("VALUES"),
								Delimiters: fsm.Delimiters{
									L: []byte{' ', '\n'},
									R: []byte{' ', '\n'},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateValuesSearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte("("),
							},
							DataHandler: fsm.DataHandlerGenericSkipToken,
						},
					},
				},
				stateTableValues: {
					NextStates: []fsm.NextState{
						{
							Name: stateTableValuesString,
							Switch: fsm.Switch{
								Trigger: []byte("'"),
							},
							DataHandler: fsm.DataHandlerGenericVoid,
						},
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte(","),
							},
							DataHandler: dhCreateTableValues,
						},
						{
							Name: stateTableValuesEnd,
							Switch: fsm.Switch{
								Trigger: []byte(")"),
							},
							DataHandler: dhCreateTableValuesEnd,
						},
					},
				},
				stateTableValuesString: {
					NextStates: []fsm.NextState{
						{
							Name: stateTableValuesStringEnd,
							Switch: fsm.Switch{
								Trigger: []byte("'"),
								Escape:  true,
							},
							DataHandler: dhCreateTableValuesString,
						},
					},
				},
				stateTableValuesStringEnd: {
					NextStates: []fsm.NextState{
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte(","),
							},
							DataHandler: fsm.DataHandlerGenericVoid,
						},
						{
							Name: stateTableValuesEnd,
							Switch: fsm.Switch{
								Trigger: []byte(")"),
							},
							DataHandler: dhCreateTableValuesStringEnd,
						},
					},
				},
				stateTableValuesEnd: {
					NextStates: []fsm.NextState{
						{
							Name: stateValuesSearch,
							Switch: fsm.Switch{
								Trigger: []byte(","),
							},
							DataHandler: nil,
						},
						{
							Name: stateSomeIntermediateState,
							Switch: fsm.Switch{
								Trigger: []byte(";"),
							},
							DataHandler: nil,
						},
					},
				},
			},
		},
	)
}

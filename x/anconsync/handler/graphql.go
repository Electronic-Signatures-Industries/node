package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/Electronic-Signatures-Industries/ancon-ipld-router-sync/x/anconsync"
	"github.com/Electronic-Signatures-Industries/ancon-ipld-router-sync/x/anconsync/codegen/graph"
	"github.com/Electronic-Signatures-Industries/ancon-ipld-router-sync/x/anconsync/codegen/graph/generated"
	"github.com/gin-gonic/gin"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/must"
	"github.com/ipld/go-ipld-prime/traversal"
)

func QueryGraphQL(s anconsync.Storage) func(*gin.Context) {
	return func(c *gin.Context) {
		cfg := generated.Config{Resolvers: &graph.Resolver{}}
		cfg.Directives.FocusedTransform = func(ctx context.Context, obj interface{},
			next graphql.Resolver,
			cid, path, previousValue, newValue string) (interface{}, error) {
			lnk, err := anconsync.ParseCidLink(cid)
			if err != nil {
				return nil, err
			}
			rootNode, err := s.Load(ipld.LinkContext{}, lnk)
			if err != nil {
				return nil, err
			}

			n, err := traversal.FocusedTransform(
				rootNode,
				datamodel.ParsePath(path),
				func(progress traversal.Progress, prev datamodel.Node) (datamodel.Node, error) {
					if progress.Path.String() == path && must.String(prev) == previousValue {
						nb := prev.Prototype().NewBuilder()
						nb.AssignString(newValue)
						return nb.Build(), nil
					}
					return nil, fmt.Errorf("not found")
				}, false)

			if err != nil {
				return nil, err
			}
			link := s.Store(ipld.LinkContext{}, n)

			jsonmodel, err := anconsync.ReadFromStore(s, link.String(), "/")
			if err != nil {
				return nil, err
			}
			next(ctx)
			return jsonmodel, nil
		}
		gql := generated.NewExecutableSchema(cfg)

		var values map[string]interface{}

		c.BindJSON(&values)

		if values["query"] == "" {
			c.JSON(400, gin.H{
				"error": fmt.Errorf("missing query").Error(),
			})
			return
		}
		variables, err := json.Marshal(values["variables"])

		if err != nil {
			c.JSON(400, gin.H{
				"error": fmt.Errorf("no JSON payload found. %v", err).Error(),
			})
			return
		}

		var vars map[string]interface{}
		json.Unmarshal(variables, &vars)
		query := values["query"].(string)
		var op string
		if values["op"] != nil {
			op = values["op"].(string)
		}

		r := c.Request.WithContext(graphql.StartOperationTrace(c.Request.Context()))
		ctx := context.WithValue(r.Context(), "dag", &anconsync.DagContractTrustedContext{
			Store: s,
		})

		ex := executor.New(gql)
		opctx, err := ex.CreateOperationContext(ctx, &graphql.RawParams{
			Query:         query,
			OperationName: op,
			Variables:     vars,
		})
		ctx = graphql.WithOperationContext(ctx, opctx)
		h, ctxh := ex.DispatchOperation(ctx, opctx)

		res := h(ctxh)

		if res.Errors != nil {
			c.JSON(400, gin.H{
				"error": res.Errors.Error(),
			})
			return
		}

		c.JSON(200, res.Data)
	}
}

// func ApplyDagContractTrusted(s Storage) func(*gin.Context) {
// 	return func(c *gin.Context) {

// 		var values map[string]interface{}

// 		c.BindJSON(&values)
// 		// if values["path"] == "" {
// 		// 	c.JSON(400, gin.H{
// 		// 		"error": fmt.Errorf("missing path").Error(),
// 		// 	})
// 		// 	return
// 		// }
// 		// if values["op"] == "" {
// 		// 	c.JSON(400, gin.H{
// 		// 		"error": fmt.Errorf("missing operation").Error(),
// 		// 	})
// 		// 	return
// 		// }
// 		if values["query"] == "" {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("missing query").Error(),
// 			})
// 			return
// 		}
// 		if values["schema_cid"] == "" {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("missing schema cid").Error(),
// 			})
// 			return
// 		}
// 		if values["datasource_cid"] == "" {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("missing payload data source cid").Error(),
// 			})
// 			return
// 		}
// 		// if values["variables"] == "" {
// 		// 	c.JSON(400, gin.H{
// 		// 		"error": fmt.Errorf("missing variables").Error(),
// 		// 	})
// 		// 	return
// 		// }

// 		gqlschema := values["schema_cid"].(string)
// 		jsonPayload := values["datasource_cid"].(string)
// 		variables, err := json.Marshal(values["variables"])

// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("no JSON payload found. %v", err).Error(),
// 			})
// 			return
// 		}
// 		query := values["query"].(string)
// 		op := values["op"].(string)

// 		// JSON Payload
// 		payload, err := ReadFromStore(s, jsonPayload, "")

// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("no JSON payload found. %v", err).Error(),
// 			})
// 			return
// 		}
// 		// GraphQL Schema
// 		schemaGQL, err := ReadFromStore(s, gqlschema, "")
// 		var v map[string]string
// 		json.Unmarshal([]byte(schemaGQL), &v)

// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("no GraphQL Schema found. %v", err).Error(),
// 			})
// 			return
// 		}
// 		schema, err := NewSchemaFrom([]byte(v["schema"]))
// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("Schema generation failed %v", err).Error(),
// 			})
// 			return
// 		}

// 		engineConf := graphql.NewEngineV2Configuration(schema)

// 		engineConf.AddDataSource(plan.DataSourceConfiguration{
// 			Custom: []byte(payload),
// 		})

// 		ctx, cancel := context.WithCancel(context.Background())
// 		defer cancel()
// 		engine, err := graphql.NewExecutionEngineV2(ctx, abstractlogger.Noop{}, engineConf)

// 		operation := graphql.Request{
// 			OperationName: op,
// 			Variables:     (variables),
// 			Query:         query,
// 		}

// 		resultWriter := graphql.NewEngineResultWriter()
// 		execCtx, execCtxCancel := context.WithCancel(context.Background())
// 		defer execCtxCancel()
// 		err = engine.Execute(execCtx, &operation, &resultWriter)
// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("Error while executing data contract transaction. %v", err).Error(),
// 			})
// 			return
// 		}

// 		// buff, _ := base64.StdEncoding.DecodeString(resultWriter.String())
// 		n, err := Decode(basicnode.Prototype.Any, resultWriter.String())
// 		if err != nil {
// 			c.JSON(400, gin.H{
// 				"error": fmt.Errorf("Decode Error %v", err).Error(),
// 			})
// 			return
// 		}
// 		cid := s.Store(ipld.LinkContext{LinkPath: ipld.ParsePath(values["path"].(string))}, n)
// 		c.JSON(201, gin.H{
// 			"cid": cid,
// 		})
// 	}
// }
// func NewSchemaFrom(schemaBytes []byte) (*graphql.Schema, error) {

// 	schemaReader := bytes.NewBuffer(schemaBytes)
// 	schema, err := graphql.NewSchemaFromReader(schemaReader)

// 	return schema, err
// }
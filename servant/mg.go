package servant

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	log "github.com/funkygao/log4go"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func (this *FunServantImpl) MgInsert(ctx *rpc.Context,
	kind string, table string, shardId int32,
	doc []byte) (r bool, appErr error) {
	profiler := this.profiler()

	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	// unmarshal inbound param
	// client json_encode, server json_decode into internal bson.M struct
	bsonDoc, err := this.unmarshalIn(doc)
	if err != nil {
		appErr = err
		profiler.do("mg.insert", ctx,
			"{kind^%s table^%s id^%d doc^%v} {err^%v r^%v}",
			kind, table, shardId,
			bsonDoc,
			appErr,
			r)

		return
	}

	// do insert and check error
	err = sess.DB().C(table).Insert(bsonDoc)
	if err != nil {
		// will not rais app error
		log.Error(err)
	} else {
		r = true
	}

	profiler.do("mg.insert", ctx,
		"{kind^%s table^%s id^%d doc^%v} {err^%v r^%v}",
		kind, table, shardId,
		bsonDoc,
		appErr,
		r)

	return
}

func (this *FunServantImpl) MgInserts(ctx *rpc.Context,
	kind string, table string, shardId int32,
	docs [][]byte) (r bool, appErr error) {
	profiler := this.profiler()

	// get mongodb session
	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	// unmarsal inbound param
	// client bson_encode, server bson_decode into internal bson.M struct
	bsonDocs := make([]interface{}, len(docs))
	for i, doc := range docs {
		bsonDoc, err := this.unmarshalIn(doc)
		if err != nil {
			appErr = err
			return
		}

		bsonDocs[i] = bsonDoc
	}

	// do insert and check error
	err = sess.DB().C(table).Insert(bsonDocs...)
	if err != nil {
		// will not rais app error
		log.Error("mg.inserts: %v", err)
	} else {
		r = true
	}

	profiler.do("mg.inserts", ctx,
		"{kind^%s table^%s id^%d docs^%d} {err^%v r^%v}",
		kind, table, shardId,
		len(docs),
		appErr,
		r)

	return
}

func (this *FunServantImpl) MgDelete(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte) (r bool, appErr error) {
	profiler := this.profiler()

	// get mongodb session
	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}
	err = sess.DB().C(table).Remove(bsonQuery)
	if err == nil {
		r = true
	}

	profiler.do("mg.del", ctx,
		"{kind^%s table^%s id^%d query^%v} {err^%v r^%v}",
		kind, table, shardId,
		bsonQuery,
		appErr,
		r)

	return
}

func (this *FunServantImpl) MgFindOne(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte, fields []byte) (r []byte,
	miss *rpc.TMongoNotFound, appErr error) {
	profiler := this.profiler()

	// get mongodb session
	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}
	var bsonFields bson.M
	if !this.mgFieldsIsNil(fields) {
		bsonFields, err = this.unmarshalIn(fields)
		if err != nil {
			appErr = err
			return
		}
	}

	var result bson.M
	q := sess.DB().C(table).Find(bsonQuery)
	if !this.mgFieldsIsNil(fields) {
		q.Select(bsonFields)
	}
	err = q.One(&result)
	if err != nil {
		if err != mgo.ErrNotFound {
			log.Error(err)
		} else {
			miss = rpc.NewTMongoNotFound()
			miss.Message = thrift.StringPtr(err.Error())
			profiler.do("mg.findOne", ctx,
				"{kind^%s table^%s id^%d query^%v fields^%v} {miss^%v err^%v val^%v}",
				kind, table, shardId,
				bsonQuery,
				bsonFields,
				miss,
				appErr,
				result)
			return
		}

		appErr = err
		return
	}

	r = this.marshalOut(result)

	profiler.do("mg.findOne", ctx,
		"{kind^%s table^%s id^%d query^%v fields^%v} {miss^%v err^%v val^%v}",
		kind, table, shardId,
		bsonQuery,
		bsonFields,
		miss,
		appErr,
		result)

	return
}

func (this *FunServantImpl) MgFindAll(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte, fields []byte, limit int32, skip int32,
	orderBy []string) (r [][]byte, appErr error) {
	profiler := this.profiler()

	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}
	bsonFields, err := this.unmarshalIn(fields)
	if err != nil {
		appErr = err
		return
	}

	q := sess.DB().C(table).Find(bsonQuery).Select(bsonFields)
	if limit > 0 {
		q.Limit(int(limit))
	}
	if skip > 0 {
		q.Skip(int(skip))
	}
	q.Sort(orderBy...)
	var result []bson.M
	err = q.All(&result)
	if err == nil {
		r = make([][]byte, len(result))
		for i, v := range result {
			r[i] = this.marshalOut(v)
		}
	} else {
		appErr = err
	}

	profiler.do("mg.findAll", ctx,
		"{kind^%s table^%s id^%d query%v fields^%v} {err^%v rl^%d}",
		kind, table, shardId,
		bsonQuery,
		bsonFields,
		appErr,
		len(r))

	return
}

func (this *FunServantImpl) MgUpdate(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte, change []byte) (r bool, appErr error) {
	profiler := this.profiler()

	// get mongodb session
	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}
	bsonChange, err := this.unmarshalIn(change)
	if err != nil {
		appErr = err
		return
	}

	err = sess.DB().C(table).Update(bsonQuery, bsonChange)
	if err == nil {
		r = true
	} else {
		log.Error("mg.update %v", err)
	}

	profiler.do("mg.update", ctx,
		"{kind^%s table^%s id^%d query^%v chg^%v} {err^%v r^%v}",
		kind, table, shardId,
		bsonQuery,
		bsonChange,
		appErr,
		r)

	return
}

func (this *FunServantImpl) MgUpdateId(ctx *rpc.Context,
	kind string, table string, shardId int32,
	id int32, change []byte) (r bool, appErr error) {
	appErr = ErrNotImplemented
	return
}

func (this *FunServantImpl) MgUpsert(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte, change []byte) (r bool, appErr error) {
	profiler := this.profiler()

	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}
	bsonChange, err := this.unmarshalIn(change)
	if err != nil {
		appErr = err
		return
	}

	_, err = sess.DB().C(table).Upsert(bsonQuery, bsonChange)
	if err == nil {
		r = true
	}

	profiler.do("mg.upsert", ctx,
		"{kind^%s table^%s id^%d query^%v chg^%v} {err^%v r^%v}",
		kind, table, shardId,
		bsonQuery,
		bsonChange,
		appErr,
		r)

	return
}

func (this *FunServantImpl) MgUpsertId(ctx *rpc.Context,
	kind string, table string, shardId int32,
	id int32, change []byte) (r bool, appErr error) {
	appErr = ErrNotImplemented
	return
}

func (this *FunServantImpl) MgCount(ctx *rpc.Context,
	kind string, table string, shardId int32,
	query []byte) (n int32, appErr error) {
	profiler := this.profiler()

	// get mongodb session
	sess, err := this.mongoSession(kind, shardId)
	if err != nil {
		appErr = err
		return
	}
	defer sess.Recyle(&err)

	bsonQuery, err := this.unmarshalIn(query)
	if err != nil {
		appErr = err
		return
	}

	var r int
	r, appErr = sess.DB().C(table).Find(bsonQuery).Count()
	n = int32(r)

	profiler.do("mg.count", ctx,
		"{kind^%s table^%s id^%d query^%v} {err^%v r^%d}",
		kind, table, shardId,
		bsonQuery,
		appErr,
		n)

	return
}

func (this *FunServantImpl) MgFindAndModify(ctx *rpc.Context,
	kind string, table string, shardId int32,
	command []byte) (r []byte, appErr error) {

	return
}

func (this *FunServantImpl) MgFindId(ctx *rpc.Context,
	kind string, table string, shardId int32,
	id []byte) (r []byte, appErr error) {
	appErr = ErrNotImplemented
	return
}

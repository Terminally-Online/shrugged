package introspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/terminally-online/shrugged/internal/parser"
)

func Database(ctx context.Context, databaseURL string) (*parser.Schema, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	schema := &parser.Schema{}

	loaders := []func(context.Context, *pgx.Conn, *parser.Schema) error{
		loadNamespaces,
		loadExtensions,
		loadEnums,
		loadDomains,
		loadCompositeTypes,
		loadSequences,
		loadTables,
		loadIndexes,
		loadViews,
		loadMaterializedViews,
		loadFunctions,
		loadProcedures,
		loadAggregates,
		loadTriggers,
		loadEventTriggers,
		loadRules,
		loadPolicies,
		loadCollations,
		loadTextSearchConfigs,
		loadPublications,
		loadSubscriptions,
		loadForeignDataWrappers,
		loadForeignServers,
		loadForeignTables,
		loadOperators,
		loadComments,
		loadRoles,
		loadRoleGrants,
		loadDefaultPrivileges,
	}

	for _, loader := range loaders {
		if err := loader(ctx, conn, schema); err != nil {
			return nil, err
		}
	}

	return schema, nil
}

func loadTables(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			c.table_schema,
			c.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			c.udt_name
		FROM information_schema.columns c
		JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema
		WHERE c.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND t.table_type = 'BASE TABLE'
		ORDER BY c.table_schema, c.table_name, c.ordinal_position
	`)
	if err != nil {
		return fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*parser.Table)

	for rows.Next() {
		var schemaName, tableName, columnName, dataType, isNullable string
		var columnDefault, udtName *string

		if err := rows.Scan(&schemaName, &tableName, &columnName, &dataType, &isNullable, &columnDefault, &udtName); err != nil {
			return fmt.Errorf("failed to scan column: %w", err)
		}

		key := schemaName + "." + tableName
		table, ok := tableMap[key]
		if !ok {
			table = &parser.Table{Schema: schemaName, Name: tableName}
			tableMap[key] = table
		}

		col := parser.Column{
			Name:     columnName,
			Type:     resolveType(dataType, udtName),
			Nullable: isNullable == "YES",
		}
		if columnDefault != nil {
			col.Default = *columnDefault
		}

		table.Columns = append(table.Columns, col)
	}

	if err := loadConstraints(ctx, conn, tableMap); err != nil {
		return err
	}

	if err := loadPartitionInfo(ctx, conn, tableMap); err != nil {
		return err
	}

	for _, table := range tableMap {
		schema.Tables = append(schema.Tables, *table)
	}

	return nil
}

func loadConstraints(ctx context.Context, conn *pgx.Conn, tableMap map[string]*parser.Table) error {
	rows, err := conn.Query(ctx, `
		SELECT
			tc.table_schema,
			tc.table_name,
			tc.constraint_name,
			tc.constraint_type,
			string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) as columns,
			ccu.table_schema AS ref_schema,
			ccu.table_name AS ref_table,
			string_agg(DISTINCT ccu.column_name, ',') AS ref_columns,
			rc.delete_rule,
			rc.update_rule,
			pg_get_constraintdef(pgc.oid) as constraint_def
		FROM information_schema.table_constraints tc
		LEFT JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		LEFT JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
			AND tc.constraint_type = 'FOREIGN KEY'
		LEFT JOIN information_schema.referential_constraints rc
			ON tc.constraint_name = rc.constraint_name
			AND tc.table_schema = rc.constraint_schema
		LEFT JOIN pg_constraint pgc
			ON pgc.conname = tc.constraint_name
			AND pgc.connamespace = (SELECT oid FROM pg_namespace WHERE nspname = tc.table_schema)
		WHERE tc.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND tc.constraint_type IN ('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE', 'CHECK')
		GROUP BY tc.table_schema, tc.table_name, tc.constraint_name, tc.constraint_type, ccu.table_schema, ccu.table_name, rc.delete_rule, rc.update_rule, pgc.oid
		ORDER BY tc.table_schema, tc.table_name, tc.constraint_name
	`)
	if err != nil {
		return fmt.Errorf("failed to query constraints: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, constraintName, constraintType string
		var columns, refSchema, refTable, refColumns, deleteRule, updateRule, constraintDef *string

		if err := rows.Scan(&schemaName, &tableName, &constraintName, &constraintType, &columns, &refSchema, &refTable, &refColumns, &deleteRule, &updateRule, &constraintDef); err != nil {
			return fmt.Errorf("failed to scan constraint: %w", err)
		}

		key := schemaName + "." + tableName
		table, ok := tableMap[key]
		if !ok {
			continue
		}

		if constraintType == "CHECK" && strings.HasSuffix(constraintName, "_not_null") {
			continue
		}

		constraint := parser.Constraint{
			Name: constraintName,
			Type: constraintType,
		}

		if columns != nil && *columns != "" {
			constraint.Columns = strings.Split(*columns, ",")
		}

		if refTable != nil {
			if refSchema != nil && *refSchema != schemaName {
				constraint.RefTable = *refSchema + "." + *refTable
			} else {
				constraint.RefTable = *refTable
			}
		}
		if refColumns != nil && *refColumns != "" {
			constraint.RefColumns = strings.Split(*refColumns, ",")
		}
		if deleteRule != nil {
			constraint.OnDelete = *deleteRule
		}
		if updateRule != nil {
			constraint.OnUpdate = *updateRule
		}
		if constraintDef != nil && constraintType == "CHECK" {
			constraint.Check = *constraintDef
		}

		table.Constraints = append(table.Constraints, constraint)
	}

	return nil
}

func loadPartitionInfo(ctx context.Context, conn *pgx.Conn, tableMap map[string]*parser.Table) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			CASE p.partstrat
				WHEN 'l' THEN 'LIST'
				WHEN 'r' THEN 'RANGE'
				WHEN 'h' THEN 'HASH'
			END AS partition_strategy,
			pg_get_partkeydef(c.oid) AS partition_key
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_partitioned_table p ON c.oid = p.partrelid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
	`)
	if err != nil {
		return fmt.Errorf("failed to query partition info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, partStrategy, partKey string
		if err := rows.Scan(&schemaName, &tableName, &partStrategy, &partKey); err != nil {
			return fmt.Errorf("failed to scan partition info: %w", err)
		}

		key := schemaName + "." + tableName
		if table, ok := tableMap[key]; ok {
			table.PartitionBy = partStrategy
			table.PartitionKey = partKey
		}
	}

	partRows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			pn.nspname AS parent_schema,
			pc.relname AS parent_table,
			pg_get_expr(c.relpartbound, c.oid) AS partition_bound
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_inherits i ON c.oid = i.inhrelid
		JOIN pg_class pc ON i.inhparent = pc.oid
		JOIN pg_namespace pn ON pc.relnamespace = pn.oid
		WHERE c.relispartition
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
	`)
	if err != nil {
		return fmt.Errorf("failed to query partition bounds: %w", err)
	}
	defer partRows.Close()

	for partRows.Next() {
		var schemaName, tableName, parentSchema, parentTable, partBound string
		if err := partRows.Scan(&schemaName, &tableName, &parentSchema, &parentTable, &partBound); err != nil {
			return fmt.Errorf("failed to scan partition bound: %w", err)
		}

		key := schemaName + "." + tableName
		if table, ok := tableMap[key]; ok {
			if parentSchema != schemaName {
				table.PartitionOf = parentSchema + "." + parentTable
			} else {
				table.PartitionOf = parentTable
			}
			table.PartitionBound = partBound
		}
	}

	return nil
}

func loadIndexes(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			i.relname AS index_name,
			t.relname AS table_name,
			ix.indisunique AS is_unique,
			am.amname AS using_method,
			pg_get_indexdef(ix.indexrelid) AS index_def,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns,
			pg_get_expr(ix.indpred, ix.indrelid) AS where_clause
		FROM pg_index ix
		JOIN pg_class i ON ix.indexrelid = i.oid
		JOIN pg_class t ON ix.indrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		JOIN pg_am am ON i.relam = am.oid
		LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND NOT ix.indisprimary
		AND NOT EXISTS (
			SELECT 1 FROM pg_constraint c
			WHERE c.conindid = ix.indexrelid AND c.contype = 'u'
		)
		GROUP BY n.nspname, i.relname, t.relname, ix.indisunique, am.amname, ix.indexrelid, ix.indpred
		ORDER BY n.nspname, t.relname, i.relname
	`)
	if err != nil {
		return fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, indexName, tableName, usingMethod, indexDef string
		var isUnique bool
		var columns []string
		var whereClause *string

		if err := rows.Scan(&schemaName, &indexName, &tableName, &isUnique, &usingMethod, &indexDef, &columns, &whereClause); err != nil {
			return fmt.Errorf("failed to scan index: %w", err)
		}

		index := parser.Index{
			Schema:     schemaName,
			Name:       indexName,
			Table:      tableName,
			Unique:     isUnique,
			Using:      usingMethod,
			Columns:    columns,
			Definition: indexDef,
		}

		if whereClause != nil {
			index.Where = *whereClause
		}

		schema.Indexes = append(schema.Indexes, index)
	}

	return nil
}

func loadViews(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT schemaname, viewname, definition
		FROM pg_views
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schemaname, viewname
	`)
	if err != nil {
		return fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, definition string
		if err := rows.Scan(&schemaName, &name, &definition); err != nil {
			return fmt.Errorf("failed to scan view: %w", err)
		}

		schema.Views = append(schema.Views, parser.View{
			Schema:     schemaName,
			Name:       name,
			Definition: strings.TrimSuffix(strings.TrimSpace(definition), ";"),
		})
	}

	return nil
}

func loadFunctions(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			p.proname,
			pg_get_function_arguments(p.oid) AS args,
			pg_get_function_result(p.oid) AS returns,
			l.lanname AS language,
			p.prosrc AS body,
			pg_get_functiondef(p.oid) AS definition
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		JOIN pg_language l ON p.prolang = l.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND p.prokind = 'f'
		ORDER BY n.nspname, p.proname
	`)
	if err != nil {
		return fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, args, returns, language, body, definition string
		if err := rows.Scan(&schemaName, &name, &args, &returns, &language, &body, &definition); err != nil {
			return fmt.Errorf("failed to scan function: %w", err)
		}

		fn := parser.Function{
			Schema:     schemaName,
			Name:       name,
			Args:       args,
			Returns:    returns,
			Language:   language,
			Body:       body,
			Definition: definition,
		}

		schema.Functions = append(schema.Functions, fn)
	}

	return nil
}

func loadTriggers(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			t.tgname,
			c.relname AS table_name,
			pg_get_triggerdef(t.oid) AS trigger_def
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND NOT t.tgisinternal
		ORDER BY n.nspname, c.relname, t.tgname
	`)
	if err != nil {
		return fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, tableName, triggerDef string
		if err := rows.Scan(&schemaName, &name, &tableName, &triggerDef); err != nil {
			return fmt.Errorf("failed to scan trigger: %w", err)
		}

		trigger := parser.Trigger{
			Schema:     schemaName,
			Name:       name,
			Table:      tableName,
			Definition: triggerDef,
		}

		schema.Triggers = append(schema.Triggers, trigger)
	}

	return nil
}

func loadSequences(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			s.schemaname,
			s.sequencename,
			s.start_value,
			s.increment_by,
			s.min_value,
			s.max_value,
			s.cache_size,
			s.cycle
		FROM pg_sequences s
		WHERE s.schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY s.schemaname, s.sequencename
	`)
	if err != nil {
		return fmt.Errorf("failed to query sequences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name string
		var start, increment, minVal, maxVal, cache *int64
		var cycle *bool

		if err := rows.Scan(&schemaName, &name, &start, &increment, &minVal, &maxVal, &cache, &cycle); err != nil {
			return fmt.Errorf("failed to scan sequence: %w", err)
		}

		seq := parser.Sequence{Schema: schemaName, Name: name}
		if start != nil {
			seq.Start = *start
		}
		if increment != nil {
			seq.Increment = *increment
		}
		if minVal != nil {
			seq.MinValue = *minVal
		}
		if maxVal != nil {
			seq.MaxValue = *maxVal
		}
		if cache != nil {
			seq.Cache = *cache
		}
		if cycle != nil {
			seq.Cycle = *cycle
		}

		schema.Sequences = append(schema.Sequences, seq)
	}

	return nil
}

func loadEnums(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			t.typname,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) AS values
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		GROUP BY n.nspname, t.typname
		ORDER BY n.nspname, t.typname
	`)
	if err != nil {
		return fmt.Errorf("failed to query enums: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name string
		var values []string
		if err := rows.Scan(&schemaName, &name, &values); err != nil {
			return fmt.Errorf("failed to scan enum: %w", err)
		}

		schema.Enums = append(schema.Enums, parser.Enum{
			Schema: schemaName,
			Name:   name,
			Values: values,
		})
	}

	return nil
}

func resolveType(dataType string, udtName *string) string {
	if udtName != nil && *udtName != "" {
		if dataType == "ARRAY" && strings.HasPrefix(*udtName, "_") {
			elementType := strings.TrimPrefix(*udtName, "_")
			resolved := resolveType(elementType, &elementType)
			return resolved + "[]"
		}

		switch *udtName {
		case "int4":
			return "integer"
		case "int8":
			return "bigint"
		case "int2":
			return "smallint"
		case "float4":
			return "real"
		case "float8":
			return "double precision"
		case "bool":
			return "boolean"
		case "timestamptz":
			return "timestamp with time zone"
		case "timestamp":
			return "timestamp without time zone"
		default:
			if dataType == "USER-DEFINED" {
				return *udtName
			}
		}
	}
	return dataType
}

func loadExtensions(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			e.extname,
			e.extversion,
			n.nspname AS schema
		FROM pg_extension e
		JOIN pg_namespace n ON e.extnamespace = n.oid
		WHERE e.extname != 'plpgsql'
		ORDER BY e.extname
	`)
	if err != nil {
		return fmt.Errorf("failed to query extensions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, version, schemaName string
		if err := rows.Scan(&name, &version, &schemaName); err != nil {
			return fmt.Errorf("failed to scan extension: %w", err)
		}

		schema.Extensions = append(schema.Extensions, parser.Extension{
			Name:    name,
			Version: version,
			Schema:  schemaName,
		})
	}

	return nil
}

func loadDomains(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			t.typname,
			pg_catalog.format_type(t.typbasetype, t.typtypmod) AS base_type,
			t.typnotnull,
			t.typdefault,
			pg_get_constraintdef(c.oid) AS check_constraint,
			col.collname AS collation
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		LEFT JOIN pg_constraint c ON c.contypid = t.oid
		LEFT JOIN pg_collation col ON t.typcollation = col.oid AND col.collname != 'default'
		WHERE t.typtype = 'd'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, t.typname
	`)
	if err != nil {
		return fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, baseType string
		var notNull bool
		var defaultVal, checkConstraint, collation *string

		if err := rows.Scan(&schemaName, &name, &baseType, &notNull, &defaultVal, &checkConstraint, &collation); err != nil {
			return fmt.Errorf("failed to scan domain: %w", err)
		}

		domain := parser.Domain{
			Schema:  schemaName,
			Name:    name,
			Type:    baseType,
			NotNull: notNull,
		}
		if defaultVal != nil {
			domain.Default = *defaultVal
		}
		if checkConstraint != nil {
			domain.Check = *checkConstraint
		}
		if collation != nil {
			domain.Collation = *collation
		}

		schema.Domains = append(schema.Domains, domain)
	}

	return nil
}

func loadCompositeTypes(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			t.typname,
			array_agg(a.attname ORDER BY a.attnum) AS attr_names,
			array_agg(pg_catalog.format_type(a.atttypid, a.atttypmod) ORDER BY a.attnum) AS attr_types
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_class c ON t.typrelid = c.oid
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
		WHERE t.typtype = 'c'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND c.relkind = 'c'
		GROUP BY n.nspname, t.typname
		ORDER BY n.nspname, t.typname
	`)
	if err != nil {
		return fmt.Errorf("failed to query composite types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name string
		var attrNames, attrTypes []string

		if err := rows.Scan(&schemaName, &name, &attrNames, &attrTypes); err != nil {
			return fmt.Errorf("failed to scan composite type: %w", err)
		}

		ct := parser.CompositeType{Schema: schemaName, Name: name}
		for i := range attrNames {
			ct.Attributes = append(ct.Attributes, parser.Column{
				Name: attrNames[i],
				Type: attrTypes[i],
			})
		}

		schema.CompositeTypes = append(schema.CompositeTypes, ct)
	}

	return nil
}

func loadMaterializedViews(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.relname,
			pg_get_viewdef(c.oid) AS definition,
			t.spcname AS tablespace,
			c.relispopulated AS with_data
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		LEFT JOIN pg_tablespace t ON c.reltablespace = t.oid
		WHERE c.relkind = 'm'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, c.relname
	`)
	if err != nil {
		return fmt.Errorf("failed to query materialized views: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, definition string
		var tablespace *string
		var withData bool

		if err := rows.Scan(&schemaName, &name, &definition, &tablespace, &withData); err != nil {
			return fmt.Errorf("failed to scan materialized view: %w", err)
		}

		mv := parser.MaterializedView{
			Schema:     schemaName,
			Name:       name,
			Definition: strings.TrimSuffix(strings.TrimSpace(definition), ";"),
			WithData:   withData,
		}
		if tablespace != nil {
			mv.Tablespace = *tablespace
		}

		schema.MaterializedViews = append(schema.MaterializedViews, mv)
	}

	return nil
}

func loadAggregates(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			p.proname,
			pg_get_function_arguments(p.oid) AS args,
			a.aggtransfn::regproc AS sfunc,
			a.aggtranstype::regtype AS stype,
			COALESCE(a.aggfinalfn::regproc::text, '-') AS finalfunc,
			a.agginitval AS initcond,
			COALESCE(a.aggsortop::regoperator::text, '0') AS sortop
		FROM pg_aggregate a
		JOIN pg_proc p ON a.aggfnoid = p.oid
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, p.proname
	`)
	if err != nil {
		return fmt.Errorf("failed to query aggregates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, args, sfunc, stype, finalfunc, sortop string
		var initcond *string

		if err := rows.Scan(&schemaName, &name, &args, &sfunc, &stype, &finalfunc, &initcond, &sortop); err != nil {
			return fmt.Errorf("failed to scan aggregate: %w", err)
		}

		agg := parser.Aggregate{
			Schema: schemaName,
			Name:   name,
			Args:   args,
			SFunc:  sfunc,
			SType:  stype,
		}
		if finalfunc != "-" {
			agg.FinalFunc = finalfunc
		}
		if initcond != nil {
			agg.InitCond = *initcond
		}
		if sortop != "0" {
			agg.SortOp = sortop
		}

		schema.Aggregates = append(schema.Aggregates, agg)
	}

	return nil
}

func loadRules(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			r.rulename,
			c.relname AS table_name,
			r.ev_type,
			r.is_instead,
			pg_get_ruledef(r.oid) AS definition
		FROM pg_rewrite r
		JOIN pg_class c ON r.ev_class = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND r.rulename != '_RETURN'
		ORDER BY n.nspname, c.relname, r.rulename
	`)
	if err != nil {
		return fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, tableName, evType, definition string
		var isInstead bool

		if err := rows.Scan(&schemaName, &name, &tableName, &evType, &isInstead, &definition); err != nil {
			return fmt.Errorf("failed to scan rule: %w", err)
		}

		eventMap := map[string]string{"1": "SELECT", "2": "UPDATE", "3": "INSERT", "4": "DELETE"}

		schema.Rules = append(schema.Rules, parser.Rule{
			Schema:     schemaName,
			Name:       name,
			Table:      tableName,
			Event:      eventMap[evType],
			DoInstead:  isInstead,
			Definition: definition,
		})
	}

	return nil
}

func loadPolicies(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			p.polname,
			c.relname AS table_name,
			p.polcmd,
			p.polpermissive,
			array_agg(r.rolname) FILTER (WHERE r.rolname IS NOT NULL) AS roles,
			pg_get_expr(p.polqual, p.polrelid) AS using_expr,
			pg_get_expr(p.polwithcheck, p.polrelid) AS with_check_expr
		FROM pg_policy p
		JOIN pg_class c ON p.polrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		LEFT JOIN pg_roles r ON r.oid = ANY(p.polroles)
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		GROUP BY n.nspname, p.polname, c.relname, p.polcmd, p.polpermissive, p.polqual, p.polwithcheck, p.polrelid
		ORDER BY n.nspname, c.relname, p.polname
	`)
	if err != nil {
		return fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, tableName, cmd string
		var permissive bool
		var roles []string
		var usingExpr, withCheckExpr *string

		if err := rows.Scan(&schemaName, &name, &tableName, &cmd, &permissive, &roles, &usingExpr, &withCheckExpr); err != nil {
			return fmt.Errorf("failed to scan policy: %w", err)
		}

		cmdMap := map[string]string{"*": "ALL", "r": "SELECT", "a": "INSERT", "w": "UPDATE", "d": "DELETE"}

		policy := parser.Policy{
			Schema:     schemaName,
			Name:       name,
			Table:      tableName,
			Command:    cmdMap[cmd],
			Permissive: permissive,
			Roles:      roles,
		}
		if usingExpr != nil {
			policy.Using = *usingExpr
		}
		if withCheckExpr != nil {
			policy.WithCheck = *withCheckExpr
		}

		schema.Policies = append(schema.Policies, policy)
	}

	return nil
}

func loadCollations(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.collname,
			c.collprovider,
			c.collcollate,
			c.collctype
		FROM pg_collation c
		JOIN pg_namespace n ON c.collnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, c.collname
	`)
	if err != nil {
		return fmt.Errorf("failed to query collations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, provider string
		var collate, ctype *string

		if err := rows.Scan(&schemaName, &name, &provider, &collate, &ctype); err != nil {
			return fmt.Errorf("failed to scan collation: %w", err)
		}

		providerMap := map[string]string{"d": "default", "c": "libc", "i": "icu"}

		coll := parser.Collation{
			Schema:   schemaName,
			Name:     name,
			Provider: providerMap[provider],
		}
		if collate != nil {
			coll.LcCollate = *collate
		}
		if ctype != nil {
			coll.LcCtype = *ctype
		}

		schema.Collations = append(schema.Collations, coll)
	}

	return nil
}

func loadTextSearchConfigs(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.cfgname,
			p.prsname AS parser
		FROM pg_ts_config c
		JOIN pg_ts_parser p ON c.cfgparser = p.oid
		JOIN pg_namespace n ON c.cfgnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, c.cfgname
	`)
	if err != nil {
		return fmt.Errorf("failed to query text search configs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, parserName string

		if err := rows.Scan(&schemaName, &name, &parserName); err != nil {
			return fmt.Errorf("failed to scan text search config: %w", err)
		}

		schema.TextSearchConfigs = append(schema.TextSearchConfigs, parser.TextSearchConfig{
			Schema: schemaName,
			Name:   name,
			Parser: parserName,
		})
	}

	return nil
}

func loadPublications(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			p.pubname,
			p.puballtables,
			p.pubinsert,
			p.pubupdate,
			p.pubdelete,
			p.pubtruncate,
			array_agg(c.relname) FILTER (WHERE c.relname IS NOT NULL) AS tables
		FROM pg_publication p
		LEFT JOIN pg_publication_rel pr ON p.oid = pr.prpubid
		LEFT JOIN pg_class c ON pr.prrelid = c.oid
		GROUP BY p.pubname, p.puballtables, p.pubinsert, p.pubupdate, p.pubdelete, p.pubtruncate
		ORDER BY p.pubname
	`)
	if err != nil {
		return fmt.Errorf("failed to query publications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var allTables, insert, update, delete, truncate bool
		var tables []string

		if err := rows.Scan(&name, &allTables, &insert, &update, &delete, &truncate, &tables); err != nil {
			return fmt.Errorf("failed to scan publication: %w", err)
		}

		pub := parser.Publication{
			Name:      name,
			AllTables: allTables,
			Tables:    tables,
		}
		if insert {
			pub.Operations = append(pub.Operations, "INSERT")
		}
		if update {
			pub.Operations = append(pub.Operations, "UPDATE")
		}
		if delete {
			pub.Operations = append(pub.Operations, "DELETE")
		}
		if truncate {
			pub.Operations = append(pub.Operations, "TRUNCATE")
		}

		schema.Publications = append(schema.Publications, pub)
	}

	return nil
}

func loadSubscriptions(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			s.subname,
			s.subenabled,
			s.subslotname,
			s.subpublications
		FROM pg_subscription s
		ORDER BY s.subname
	`)
	if err != nil {
		return fmt.Errorf("failed to query subscriptions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var enabled bool
		var slotName *string
		var publications []string

		if err := rows.Scan(&name, &enabled, &slotName, &publications); err != nil {
			return fmt.Errorf("failed to scan subscription: %w", err)
		}

		sub := parser.Subscription{
			Name:    name,
			Enabled: enabled,
		}
		if slotName != nil {
			sub.SlotName = *slotName
		}
		if len(publications) > 0 {
			sub.Publication = strings.Join(publications, ", ")
		}

		schema.Subscriptions = append(schema.Subscriptions, sub)
	}

	return nil
}

func loadForeignDataWrappers(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			f.fdwname,
			COALESCE(h.proname, '') AS handler,
			COALESCE(v.proname, '') AS validator,
			COALESCE(f.fdwoptions, '{}') AS options
		FROM pg_foreign_data_wrapper f
		LEFT JOIN pg_proc h ON f.fdwhandler = h.oid
		LEFT JOIN pg_proc v ON f.fdwvalidator = v.oid
		WHERE f.fdwname NOT IN ('file_fdw', 'postgres_fdw')
		   OR EXISTS (SELECT 1 FROM pg_foreign_server s WHERE s.srvfdw = f.oid)
		ORDER BY f.fdwname
	`)
	if err != nil {
		return fmt.Errorf("failed to query foreign data wrappers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, handler, validator string
		var options []string

		if err := rows.Scan(&name, &handler, &validator, &options); err != nil {
			return fmt.Errorf("failed to scan foreign data wrapper: %w", err)
		}

		fdw := parser.ForeignDataWrapper{
			Name:      name,
			Handler:   handler,
			Validator: validator,
			Options:   parseOptions(options),
		}

		schema.ForeignDataWrappers = append(schema.ForeignDataWrappers, fdw)
	}

	return nil
}

func loadForeignServers(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			s.srvname,
			f.fdwname,
			COALESCE(s.srvtype, '') AS server_type,
			COALESCE(s.srvversion, '') AS server_version,
			COALESCE(s.srvoptions, '{}') AS options
		FROM pg_foreign_server s
		JOIN pg_foreign_data_wrapper f ON s.srvfdw = f.oid
		ORDER BY s.srvname
	`)
	if err != nil {
		return fmt.Errorf("failed to query foreign servers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, fdw, serverType, serverVersion string
		var options []string

		if err := rows.Scan(&name, &fdw, &serverType, &serverVersion, &options); err != nil {
			return fmt.Errorf("failed to scan foreign server: %w", err)
		}

		server := parser.ForeignServer{
			Name:    name,
			FDW:     fdw,
			Type:    serverType,
			Version: serverVersion,
			Options: parseOptions(options),
		}

		schema.ForeignServers = append(schema.ForeignServers, server)
	}

	return nil
}

func loadForeignTables(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			c.relname,
			s.srvname,
			COALESCE(ft.ftoptions, '{}') AS options
		FROM pg_foreign_table ft
		JOIN pg_class c ON ft.ftrelid = c.oid
		JOIN pg_foreign_server s ON ft.ftserver = s.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, c.relname
	`)
	if err != nil {
		return fmt.Errorf("failed to query foreign tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, server string
		var options []string

		if err := rows.Scan(&schemaName, &name, &server, &options); err != nil {
			return fmt.Errorf("failed to scan foreign table: %w", err)
		}

		ft := parser.ForeignTable{
			Schema:  schemaName,
			Name:    name,
			Server:  server,
			Options: parseOptions(options),
		}

		schema.ForeignTables = append(schema.ForeignTables, ft)
	}

	return nil
}

func loadOperators(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			o.oprname,
			COALESCE(lt.typname, 'NONE') AS left_type,
			COALESCE(rt.typname, 'NONE') AS right_type,
			rest.typname AS result_type,
			p.proname AS procedure,
			COALESCE(com.oprname, '') AS commutator,
			COALESCE(neg.oprname, '') AS negator
		FROM pg_operator o
		JOIN pg_namespace n ON o.oprnamespace = n.oid
		LEFT JOIN pg_type lt ON o.oprleft = lt.oid
		LEFT JOIN pg_type rt ON o.oprright = rt.oid
		JOIN pg_type rest ON o.oprresult = rest.oid
		JOIN pg_proc p ON o.oprcode = p.oid
		LEFT JOIN pg_operator com ON o.oprcom = com.oid
		LEFT JOIN pg_operator neg ON o.oprnegate = neg.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY n.nspname, o.oprname, lt.typname, rt.typname
	`)
	if err != nil {
		return fmt.Errorf("failed to query operators: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, leftType, rightType, resultType, procedure, commutator, negator string

		if err := rows.Scan(&schemaName, &name, &leftType, &rightType, &resultType, &procedure, &commutator, &negator); err != nil {
			return fmt.Errorf("failed to scan operator: %w", err)
		}

		op := parser.Operator{
			Schema:     schemaName,
			Name:       name,
			LeftType:   leftType,
			RightType:  rightType,
			ResultType: resultType,
			Procedure:  procedure,
			Commutator: commutator,
			Negator:    negator,
		}

		schema.Operators = append(schema.Operators, op)
	}

	return nil
}

func parseOptions(options []string) map[string]string {
	result := make(map[string]string)
	for _, opt := range options {
		parts := strings.SplitN(opt, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func loadNamespaces(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname,
			r.rolname AS owner
		FROM pg_namespace n
		JOIN pg_roles r ON n.nspowner = r.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'pg_temp_1', 'pg_toast_temp_1', 'public')
		AND n.nspname NOT LIKE 'pg_temp_%'
		AND n.nspname NOT LIKE 'pg_toast_temp_%'
		ORDER BY n.nspname
	`)
	if err != nil {
		return fmt.Errorf("failed to query namespaces: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, owner string
		if err := rows.Scan(&name, &owner); err != nil {
			return fmt.Errorf("failed to scan namespace: %w", err)
		}

		schema.Namespaces = append(schema.Namespaces, parser.Namespace{
			Name:  name,
			Owner: owner,
		})
	}

	return nil
}

func loadProcedures(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			p.proname,
			pg_get_function_arguments(p.oid) AS args,
			l.lanname AS language,
			p.prosrc AS body,
			pg_get_functiondef(p.oid) AS definition
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		JOIN pg_language l ON p.prolang = l.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND p.prokind = 'p'
		ORDER BY n.nspname, p.proname
	`)
	if err != nil {
		return fmt.Errorf("failed to query procedures: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, name, args, language, body, definition string
		if err := rows.Scan(&schemaName, &name, &args, &language, &body, &definition); err != nil {
			return fmt.Errorf("failed to scan procedure: %w", err)
		}

		schema.Procedures = append(schema.Procedures, parser.Procedure{
			Schema:     schemaName,
			Name:       name,
			Args:       args,
			Language:   language,
			Body:       body,
			Definition: definition,
		})
	}

	return nil
}

func loadEventTriggers(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			e.evtname,
			e.evtevent,
			p.proname AS function_name,
			e.evtenabled,
			COALESCE(e.evttags, '{}') AS tags
		FROM pg_event_trigger e
		JOIN pg_proc p ON e.evtfoid = p.oid
		ORDER BY e.evtname
	`)
	if err != nil {
		return fmt.Errorf("failed to query event triggers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, event, function, enabled string
		var tags []string

		if err := rows.Scan(&name, &event, &function, &enabled, &tags); err != nil {
			return fmt.Errorf("failed to scan event trigger: %w", err)
		}

		enabledMap := map[string]string{"O": "ORIGIN", "R": "REPLICA", "A": "ALWAYS", "D": "DISABLED"}

		schema.EventTriggers = append(schema.EventTriggers, parser.EventTrigger{
			Name:     name,
			Event:    event,
			Function: function,
			Enabled:  enabledMap[enabled],
			Tags:     tags,
		})
	}

	return nil
}

func loadComments(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			CASE c.relkind
				WHEN 'r' THEN 'TABLE'
				WHEN 'v' THEN 'VIEW'
				WHEN 'm' THEN 'MATERIALIZED VIEW'
				WHEN 'i' THEN 'INDEX'
				WHEN 'S' THEN 'SEQUENCE'
				WHEN 'f' THEN 'FOREIGN TABLE'
				ELSE 'TABLE'
			END AS object_type,
			n.nspname AS schema_name,
			c.relname AS object_name,
			d.description
		FROM pg_description d
		JOIN pg_class c ON d.objoid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE d.objsubid = 0
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')

		UNION ALL

		SELECT
			'COLUMN' AS object_type,
			n.nspname AS schema_name,
			c.relname || '.' || a.attname AS object_name,
			d.description
		FROM pg_description d
		JOIN pg_class c ON d.objoid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		JOIN pg_attribute a ON d.objoid = a.attrelid AND d.objsubid = a.attnum
		WHERE d.objsubid > 0
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')

		UNION ALL

		SELECT
			'FUNCTION' AS object_type,
			n.nspname AS schema_name,
			p.proname AS object_name,
			d.description
		FROM pg_description d
		JOIN pg_proc p ON d.objoid = p.oid
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')

		UNION ALL

		SELECT
			'TYPE' AS object_type,
			n.nspname AS schema_name,
			t.typname AS object_name,
			d.description
		FROM pg_description d
		JOIN pg_type t ON d.objoid = t.oid
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND t.typtype IN ('e', 'c', 'd')

		UNION ALL

		SELECT
			'SCHEMA' AS object_type,
			'' AS schema_name,
			n.nspname AS object_name,
			d.description
		FROM pg_description d
		JOIN pg_namespace n ON d.objoid = n.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')

		ORDER BY object_type, schema_name, object_name
	`)
	if err != nil {
		return fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var objectType, schemaName, objectName, description string
		if err := rows.Scan(&objectType, &schemaName, &objectName, &description); err != nil {
			return fmt.Errorf("failed to scan comment: %w", err)
		}

		comment := parser.Comment{
			ObjectType: objectType,
			Schema:     schemaName,
			Comment:    description,
		}

		if objectType == "COLUMN" {
			parts := strings.SplitN(objectName, ".", 2)
			if len(parts) == 2 {
				comment.Name = parts[0]
				comment.Column = parts[1]
			} else {
				comment.Name = objectName
			}
		} else {
			comment.Name = objectName
		}

		schema.Comments = append(schema.Comments, comment)
	}

	return nil
}

func loadRoles(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			r.rolname,
			r.rolsuper,
			r.rolcreatedb,
			r.rolcreaterole,
			r.rolinherit,
			r.rolcanlogin,
			r.rolreplication,
			r.rolbypassrls,
			r.rolconnlimit,
			COALESCE(r.rolvaliduntil::text, '') AS valid_until,
			COALESCE(array_agg(m.rolname) FILTER (WHERE m.rolname IS NOT NULL), '{}') AS member_of
		FROM pg_roles r
		LEFT JOIN pg_auth_members am ON r.oid = am.member
		LEFT JOIN pg_roles m ON am.roleid = m.oid
		WHERE r.rolname NOT LIKE 'pg_%'
		AND r.rolname NOT IN ('postgres')
		GROUP BY r.oid, r.rolname, r.rolsuper, r.rolcreatedb, r.rolcreaterole,
				 r.rolinherit, r.rolcanlogin, r.rolreplication, r.rolbypassrls,
				 r.rolconnlimit, r.rolvaliduntil
		ORDER BY r.rolname
	`)
	if err != nil {
		return fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, validUntil string
		var superUser, createDB, createRole, inherit, login, replication, bypassRLS bool
		var connLimit int
		var inRoles []string

		if err := rows.Scan(&name, &superUser, &createDB, &createRole, &inherit, &login,
			&replication, &bypassRLS, &connLimit, &validUntil, &inRoles); err != nil {
			return fmt.Errorf("failed to scan role: %w", err)
		}

		schema.Roles = append(schema.Roles, parser.Role{
			Name:            name,
			SuperUser:       superUser,
			CreateDB:        createDB,
			CreateRole:      createRole,
			Inherit:         inherit,
			Login:           login,
			Replication:     replication,
			BypassRLS:       bypassRLS,
			ConnectionLimit: connLimit,
			ValidUntil:      validUntil,
			InRoles:         inRoles,
		})
	}

	return nil
}

func loadRoleGrants(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			'TABLE' AS object_type,
			privilege_type,
			table_schema,
			table_name,
			grantee,
			is_grantable
		FROM information_schema.table_privileges
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		AND grantor != grantee

		UNION ALL

		SELECT
			'FUNCTION' AS object_type,
			privilege_type,
			routine_schema AS table_schema,
			routine_name AS table_name,
			grantee,
			is_grantable
		FROM information_schema.routine_privileges
		WHERE routine_schema NOT IN ('pg_catalog', 'information_schema')
		AND grantor != grantee

		UNION ALL

		SELECT
			'TYPE' AS object_type,
			REPLACE(privilege_type, 'TYPE ', '') AS privilege_type,
			udt_schema AS table_schema,
			udt_name AS table_name,
			grantee,
			is_grantable
		FROM information_schema.udt_privileges
		WHERE udt_schema NOT IN ('pg_catalog', 'information_schema')
		AND grantor != grantee

		ORDER BY table_schema, table_name, grantee, privilege_type
	`)
	if err != nil {
		return fmt.Errorf("failed to query role grants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var objectType, privilege, schemaName, objectName, grantee, isGrantable string
		if err := rows.Scan(&objectType, &privilege, &schemaName, &objectName, &grantee, &isGrantable); err != nil {
			return fmt.Errorf("failed to scan role grant: %w", err)
		}

		schema.RoleGrants = append(schema.RoleGrants, parser.RoleGrant{
			ObjectType: objectType,
			Privilege:  privilege,
			Schema:     schemaName,
			ObjectName: objectName,
			Grantee:    grantee,
			WithGrant:  isGrantable == "YES",
		})
	}

	return nil
}

func loadDefaultPrivileges(ctx context.Context, conn *pgx.Conn, schema *parser.Schema) error {
	rows, err := conn.Query(ctx, `
		SELECT
			n.nspname AS schema_name,
			r.rolname AS role_name,
			CASE d.defaclobjtype
				WHEN 'r' THEN 'TABLES'
				WHEN 'S' THEN 'SEQUENCES'
				WHEN 'f' THEN 'FUNCTIONS'
				WHEN 'T' THEN 'TYPES'
				WHEN 'n' THEN 'SCHEMAS'
			END AS object_type,
			g.rolname AS grantee,
			array_agg(
				CASE a.privilege_type
					WHEN 'r' THEN 'SELECT'
					WHEN 'w' THEN 'UPDATE'
					WHEN 'a' THEN 'INSERT'
					WHEN 'd' THEN 'DELETE'
					WHEN 'D' THEN 'TRUNCATE'
					WHEN 'x' THEN 'REFERENCES'
					WHEN 't' THEN 'TRIGGER'
					WHEN 'X' THEN 'EXECUTE'
					WHEN 'U' THEN 'USAGE'
					WHEN 'C' THEN 'CREATE'
					ELSE a.privilege_type
				END
			) AS privileges
		FROM pg_default_acl d
		JOIN pg_namespace n ON d.defaclnamespace = n.oid
		JOIN pg_roles r ON d.defaclrole = r.oid
		CROSS JOIN LATERAL aclexplode(d.defaclacl) a
		JOIN pg_roles g ON a.grantee = g.oid
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		GROUP BY n.nspname, r.rolname, d.defaclobjtype, g.rolname
		ORDER BY n.nspname, r.rolname, object_type, g.rolname
	`)
	if err != nil {
		return fmt.Errorf("failed to query default privileges: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, roleName, objectType, grantee string
		var privileges []string

		if err := rows.Scan(&schemaName, &roleName, &objectType, &grantee, &privileges); err != nil {
			return fmt.Errorf("failed to scan default privilege: %w", err)
		}

		schema.DefaultPrivileges = append(schema.DefaultPrivileges, parser.DefaultPrivilege{
			Schema:     schemaName,
			Role:       roleName,
			ObjectType: objectType,
			Privileges: privileges,
			Grantee:    grantee,
		})
	}

	return nil
}

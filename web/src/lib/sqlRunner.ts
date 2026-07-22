// Исполнение SQL-запросов пользователя в браузере через sql.js (SQLite в WASM).
// WASM-файл берётся из локального ассета (никаких CDN). sql.js инициализируется
// лениво — только при первом запуске sql-задачи (этот модуль импортируется
// динамически из страницы задачи).

import initSqlJs, { type Database, type SqlJsStatic, type SqlValue } from 'sql.js'
import wasmUrl from 'sql.js/dist/sql-wasm.wasm?url'

let sqlPromise: Promise<SqlJsStatic> | null = null

function loadSql(): Promise<SqlJsStatic> {
  if (!sqlPromise) {
    sqlPromise = initSqlJs({ locateFile: () => wasmUrl })
  }
  return sqlPromise
}

export interface SqlRunResult {
  columns: string[]
  rows: string[][]
  rowCount: number
  elapsedMs: number
}

function cellToString(value: SqlValue): string {
  if (value === null) return 'NULL'
  if (value instanceof Uint8Array) return '[blob]'
  return String(value)
}

/**
 * Создаёт свежую in-memory БД из schema + seed на каждый вызов, выполняет запрос
 * пользователя и возвращает последний результирующий набор (обычно единственный SELECT).
 * Ошибки SQL пробрасываются как обычные Error с текстом от SQLite.
 */
export async function runSqlQuery(
  schema: string,
  seed: string,
  query: string,
): Promise<SqlRunResult> {
  const SQL = await loadSql()
  const db: Database = new SQL.Database()
  try {
    if (schema.trim()) db.run(schema)
    if (seed.trim()) db.run(seed)

    const started = performance.now()
    const results = db.exec(query)
    const elapsedMs = Math.max(0, Math.round(performance.now() - started))

    const last = results.length > 0 ? results[results.length - 1] : null
    const columns = last ? last.columns.map(String) : []
    const rows = last ? last.values.map((row) => row.map(cellToString)) : []

    return { columns, rows, rowCount: rows.length, elapsedMs }
  } finally {
    db.close()
  }
}

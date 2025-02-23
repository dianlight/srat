//import type { ConfigFile } from '@rtk-query/codegen-openapi'

const config /*: ConfigFile*/ = {
  schemaFile: "../backend/src/docs/swagger.json",
  apiFile: "./src/store/emptyApi.ts",
  apiImport: "emptySplitApi",
  outputFile: "./src/store/sratApi.ts",
  exportName: "sratApi",
  hooks: true,
  useEnumType: true,
  tag: true,
  mergeReadWriteOnly: true,
  includeDefault: true,
  unionUndefined: false,
}

export default config
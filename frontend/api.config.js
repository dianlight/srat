/** @type {import('@rtk-query/codegen-openapi').ConfigFile} */
const config = {
	"schemaFile": "../backend/docs/openapi.json",
	"apiFile": "./src/store/emptyApi.ts",
	"apiImport": "emptySplitApi",
	"outputFile": "./src/store/sratApi.ts",
	"exportName": "sratApi",
	"hooks": true,
	"useEnumType": true,
	"tag": true,
	"mergeReadWriteOnly": true,
	"includeDefault": true,
	"unionUndefined": false,
	"endpointOverrides": [
		{
			"pattern": /.*/,
			"parameterFilter": (name, parameter) => {
				// Exclude 'X-Span-Id' and 'X-Trace-Id' headers
				//console.log(`Should exclude ${name} in ${parameter.in}:`, (["X-Span-Id", "X-Trace-Id"].includes(name) && parameter.in === 'header'));
				return !(["X-Span-Id", "X-Trace-Id"].includes(name) && parameter.in === 'header')
			},
		}
	]
}

export default config

import type { MetaRecord } from 'nextra'

/**
 * type MetaRecordValue =
 *  | TitleSchema
 *  | PageItemSchema
 *  | SeparatorSchema
 *  | MenuSchema
 *
 * type MetaRecord = Record<string, MetaRecordValue>
 **/
const meta: MetaRecord = {
    "index": "Introduction",
    "getting-started": "Getting Started",
    "routing": "Routing",
    "middleware": "Middleware",
    "openapi": "OpenAPI Integration",
    "streaming": "Streaming (WS & SSE)",
    "testing": "Testing",
    "examples": "Examples",
    "api-reference": "API Reference",
    "migration": "Migration Guide",
    "troubleshooting": "Troubleshooting"
}

export default meta
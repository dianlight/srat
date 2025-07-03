# SRAT Companion - Home Assistant Integration

A comprehensive Home Assistant integration for SRAT services with multiple discovery methods and real-time event handling.

## Features

- **Multiple Discovery Methods:**
  - Zeroconf discovery for `_srat._tcp` services
  - Addon discovery (configurable addon slugs)
  - Manual IP/Port configuration via HA UI

- **Real-time Events:**
  - SSE (Server-Sent Events) client connection
  - Automatic reconnection with repair alerts on failure
  - Event storage and processing

- **OpenAPI Client Generation:**
  - Automatic client generation from `../../docs/openapi.yaml`
  - Uses `openapi-python-client` for type-safe API interactions

## Installation

1. Copy the `custom_components/srat_companion` directory to your Home Assistant `custom_components` folder
2. Install the required dependency:
   ```bash
   pip install openapi-python-client
   ```
3. Generate the OpenAPI client:
   ```bash
   python scripts/generate_client.py
   ```
4. Restart Home Assistant
5. Go to Configuration > Integrations > Add Integration > SRAT Companion

## Configuration

### Zeroconf Discovery
The integration automatically discovers SRAT services broadcasting on `_srat._tcp.local.`

### Addon Discovery
The integration automatically searches for running addons with these slugs:
- `srat_addon`
- `srat_core`
- `srat_main`
- `srat_server`

If any of these addons are running, they will be discovered and available for selection.

### Manual Configuration
Enter the IP address and port of your SRAT service directly.

## Components

### Sensors
- **Connection Status**: Shows if the SRAT service is connected
- **Event Count**: Number of events received from the service

### Repair Integration
Automatic repair alerts are created when SSE connection fails, providing:
- Error details
- Connection information
- Troubleshooting guidance

## API Endpoints Expected

The integration expects the following endpoints on your SRAT service:

- `GET /api/health` - Health check for connection validation
- `GET /api/status` - Service status information
- `GET /api/events` - SSE endpoint for real-time events

## Event Processing

Events received via SSE are processed and stored in the coordinator. The integration:
- Maintains the last 100 events in memory
- Notifies all listening entities when new events arrive
- Provides event data through sensor attributes

## Addon Discovery Implementation

To complete the addon discovery feature, you'll need to implement the Home Assistant Supervisor API calls:

```python
# In config_flow.py, replace the addon_discovery_not_implemented error with:
async def _discover_addons(self, addon_slugs: list[str]) -> dict[str, dict[str, Any]]:
    """Discover running addons by slug."""
    discovered = {}
    
    for slug in addon_slugs:
        try:
            # Call HA Supervisor API to check if addon is running
            # This requires accessing the supervisor API
            addon_info = await self.hass.async_add_executor_job(
                self._get_addon_info, slug
            )
            if addon_info and addon_info.get("state") == "started":
                discovered[slug] = {
                    "host": addon_info.get("ip_address"),
                    "port": addon_info.get("port", DEFAULT_PORT),
                    "name": addon_info.get("name"),
                }
        except Exception as err:
            _LOGGER.debug("Could not discover addon %s: %s", slug, err)
    
    return discovered
```

## Development

### File Structure
```
custom_components/srat_companion/
├── __init__.py          # Integration setup
├── config_flow.py       # Configuration flow
├── const.py            # Constants
├── coordinator.py      # Data coordinator with SSE
├── sensor.py           # Sensor platform
├── manifest.json       # Integration manifest
├── strings.json        # Translations
└── client/            # Generated OpenAPI client
```

### OpenAPI Client
The integration uses `openapi-python-client` to generate a type-safe client from the OpenAPI specification. Run the generation script after updating the API spec:

```bash
python scripts/generate_client.py
```

### Testing

Create a test SRAT service for development:

```python
# test_srat_server.py
from flask import Flask, Response
import json
import time

app = Flask(__name__)

@app.route('/api/health')
def health():
    return {'status': 'ok'}

@app.route('/api/status')
def status():
    return {'connected': True, 'version': '1.0.0'}

@app.route('/api/events')
def events():
    def generate():
        while True:
            event_data = {
                'timestamp': time.time(),
                'type': 'test_event',
                'data': {'counter': int(time.time()) % 100}
            }
            yield f"data: {json.dumps(event_data)}\n\n"
            time.sleep(5)
    
    return Response(generate(), mimetype='text/plain')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

### Logging

Enable debug logging in Home Assistant:

```yaml
# configuration.yaml
logger:
  default: warning
  logs:
    custom_components.srat_companion: debug
```

## Requirements

- Home Assistant 2023.1+
- Python 3.10+
- aiohttp-sse-client

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Update the OpenAPI spec if needed
5. Regenerate the client
6. Test your changes
7. Submit a pull request

## Troubleshooting

### Common Issues

1. **Connection Failed**: Check if the SRAT service is running and accessible
2. **No Services Found**: Ensure zeroconf is working and service is broadcasting
3. **SSE Connection Drops**: Check network stability and service health
4. **Addon Discovery Not Working**: Implement the supervisor API calls as shown above

### Debug Steps

1. Enable debug logging
2. Check the Home Assistant logs for detailed error messages
3. Verify the SRAT service endpoints are responding
4. Test the SSE endpoint manually with curl or a browser

## Future Enhancements

- Add more sensor types based on event data
- Implement device triggers for specific events
- Add support for multiple SRAT services
- Implement service calls for controlling SRAT
- Add configuration options for event filtering
# V2Ray Tunnel Troubleshooting Guide

## Common Error: "Failed to start V2Ray tunnel: incomplete tunnel configuration"

This error occurs when the V2Ray tunnel configuration is missing required fields. Here's how to fix it:

### Required Fields for All V2Ray Protocols

1. **Protocol** - Must be one of: `vmess`, `vless`, or `trojan`
2. **Remote Address** - The IP address or domain of your V2Ray server
3. **Remote Port** - The port number your V2Ray server is listening on
4. **Security** - Must be one of: `tls`, `reality`, or `none`
5. **Network** - Must be one of: `tcp`, `ws`, or `grpc`

### Protocol-Specific Required Fields

#### For VMess and VLESS Protocols:
- **UUID** - The unique identifier for your V2Ray user

#### For Trojan Protocol:
- **Password** - The password for your Trojan server

### Step-by-Step Configuration Guide

1. **Select Tunnel Type**: Choose "WireGuard ➜ V2Ray" from the tunnel type dropdown

2. **Fill Basic Information**:
   - **Name**: Give your tunnel a descriptive name
   - **Description**: Optional description of the tunnel's purpose

3. **Configure V2Ray Settings**:
   - **Protocol**: Select your V2Ray protocol (VMess/VLESS/Trojan)
   - **Remote Address**: Enter your V2Ray server's IP or domain
   - **Remote Port**: Enter your V2Ray server's port (e.g., 443, 8080)
   - **Security**: Choose the security method (TLS/Reality/None)
   - **Network**: Select the network type (TCP/WebSocket/gRPC)

4. **Protocol-Specific Configuration**:
   - **For VMess/VLESS**: Enter the UUID from your V2Ray server configuration
   - **For Trojan**: Enter the password from your Trojan server configuration

5. **Optional Advanced Settings**:
   - **Server Name**: For TLS connections (usually same as Remote Address)
   - **Fingerprint**: Browser fingerprint for TLS
   - **Path**: For WebSocket/gRPC connections
   - **SNI**: Server Name Indication for TLS
   - **ALPN**: Application-Layer Protocol Negotiation
   - **Flow**: For VLESS protocol (optional)

### Using V2Ray Links

You can also use the "Parse" feature to automatically fill configuration from V2Ray links:

1. **Copy your V2Ray link** (vmess://, vless://, or trojan://)
2. **Paste it into the link field** in the V2Ray configuration section
3. **Click "Parse"** to automatically fill all fields
4. **Click "Validate"** to check if all required fields are filled

### Validation Tips

- Use the **"Validate"** button to check your configuration before creating the tunnel
- Required fields are marked with red asterisks (*)
- The validation will show specific errors for missing fields
- Make sure your V2Ray server is running and accessible

### Common Issues and Solutions

#### Issue: "UUID is required for vmess protocol"
**Solution**: Enter the UUID from your V2Ray server configuration. This is typically a long string of characters like `12345678-1234-1234-1234-123456789abc`.

#### Issue: "Password is required for Trojan protocol"
**Solution**: Enter the password configured on your Trojan server.

#### Issue: "V2Ray remote address is missing"
**Solution**: Enter the IP address or domain name of your V2Ray server.

#### Issue: "V2Ray remote port is missing"
**Solution**: Enter the port number your V2Ray server is listening on (e.g., 443, 8080, 8443).

#### Issue: "V2Ray security setting is missing"
**Solution**: Select a security method from the dropdown (TLS, Reality, or None).

#### Issue: "V2Ray network type is missing"
**Solution**: Select a network type from the dropdown (TCP, WebSocket, or gRPC).

### Testing Your Configuration

1. **Fill all required fields** as described above
2. **Click "Validate"** to check for errors
3. **Fix any validation errors** that appear
4. **Create the tunnel** when validation passes
5. **Start the tunnel** and check the logs for any runtime errors

### Getting Help

If you continue to experience issues:

1. **Check the logs** in the tunnel management interface
2. **Verify your V2Ray server** is running and accessible
3. **Test your V2Ray configuration** with a standard V2Ray client first
4. **Ensure all required fields** are properly filled
5. **Check network connectivity** to your V2Ray server

### Example Valid Configurations

#### VMess Configuration:
```
Protocol: vmess
Remote Address: example.com
Remote Port: 443
UUID: 12345678-1234-1234-1234-123456789abc
Security: tls
Network: tcp
```

#### Trojan Configuration:
```
Protocol: trojan
Remote Address: example.com
Remote Port: 443
Password: your_trojan_password
Security: tls
Network: tcp
```

#### VLESS Configuration:
```
Protocol: vless
Remote Address: example.com
Remote Port: 443
UUID: 12345678-1234-1234-1234-123456789abc
Security: reality
Network: grpc
Path: grpc
``` 
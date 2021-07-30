const go = new Go();
WebAssembly.instantiateStreaming( fetch( keyManager.wasm ), go.importObject ).then( ( result ) =>
    go.run( result.instance )
);
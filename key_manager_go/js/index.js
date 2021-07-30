import('./node_modules/key_manager/go_key_manager').then((js) => {
    console.log('init done');
    window.genKey = js.GenerateKeyPair;
});

import Service from '@ember/service';

// promise aware assertion
export const ifNotBlocking = function(repo) {
  return Promise.resolve(repo.get().refresh === 'off');
};

export default class UiConfigService extends Service {
  config = undefined;

  get() {
    if (this.config === undefined) {
      // Load config from our special meta tag for now. Later it might come from
      // an API instead/as well.
      var meta = unescape(document.getElementsByName('consul-ui/ui_config')[0].content);
      this.config = JSON.parse(meta);
    }
    return this.config;
  }
}

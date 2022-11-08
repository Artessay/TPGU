module.exports = {
  devServer: {
    proxy: {
      '/query': {
        target: 'https://backend.bnva.projects.zjvis.org',
        ws: true,
        changeOrigin: true
      }
    }
  }
}

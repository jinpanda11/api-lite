// Fix: @douyinfe/semi-icons root index.d.ts does not re-export individual icons.
// This augments the module so named icon imports work correctly.
declare module '@douyinfe/semi-icons' {
  export * from '@douyinfe/semi-icons/lib/es/icons'
}

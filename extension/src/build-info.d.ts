// Injected by build.mjs via esbuild's `define` -- a compile-time
// constant, not a runtime lookup. See build.mjs's own buildInfo()
// for the exact format ("build <short-rev> · <UTC time>"), matching
// deploy/deploy.sh's ${PERSONA_BUILD_INFO} convention for the public
// site.
declare const __BUILD_INFO__: string;

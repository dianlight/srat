## CI Build Failure Issue

### Description
There is a CI build failure occurring due to incorrect usage of the `Provider` component in several test files. The error is caused by using the `Provider` without the required `children` prop.

### Affected Test Files
- `src/pages/dashboard/__tests__/DashboardMetrics.test.tsx`
- `src/pages/volumes/components/__tests__/VolumeMountDialog.test.tsx`
- `src/pages/volumes/components/__tests__/VolumesTreeView.test.tsx`

### Error Details
The error logs from job [#18358705772](https://github.com/dianlight/srat/actions/runs/18358705772/job/52297189630) indicate that the `Provider` must wrap the tested components as children. 

### Proposed Fix
Change `
<Provider store={store} />
` to `
<Provider store={store}>{/* component(s) */}</Provider>
` in the affected test files. This will resolve the 'Property children is missing' TypeScript error and allow tests to pass.

### Next Steps
- Review the affected test files and implement the proposed fix to prevent CI build failures.
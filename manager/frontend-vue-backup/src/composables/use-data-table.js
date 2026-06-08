import { ref, reactive } from 'vue'

/**
 * Encapsulates pagination + sort state for server-side DataTable usage.
 * Pass the returned state to DataTable's @sort-change and @page-change handlers.
 */
export function useDataTable({ initialPageSize = 20 } = {}) {
  const pagination = reactive({ pageIndex: 0, pageSize: initialPageSize })
  const sorting = ref([])
  const loading = ref(false)

  function onSortChange(newSorting) {
    sorting.value = newSorting
    pagination.pageIndex = 0 // reset to first page on sort
  }

  function onPageChange(newPagination) {
    pagination.pageIndex = newPagination.pageIndex
    pagination.pageSize = newPagination.pageSize
  }

  function reset() {
    pagination.pageIndex = 0
    sorting.value = []
  }

  return { pagination, sorting, loading, onSortChange, onPageChange, reset }
}

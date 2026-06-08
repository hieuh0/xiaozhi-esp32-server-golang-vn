<script setup>
import { ref, computed } from 'vue'
import {
  useVueTable,
  getCoreRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  getFilteredRowModel,
  FlexRender
} from '@tanstack/vue-table'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const props = defineProps({
  columns: { type: Array, required: true },
  data: { type: Array, required: true },
  pageSize: { type: Number, default: 20 },
  /** Set true to disable built-in pagination and emit events for server-side handling */
  serverSide: { type: Boolean, default: false },
  /** Total row count for server-side mode */
  total: { type: Number, default: 0 },
  /** Show a global filter input */
  searchable: { type: Boolean, default: false },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['sort-change', 'page-change'])

const globalFilter = ref('')
const sorting = ref([])
const pagination = ref({ pageIndex: 0, pageSize: props.pageSize })

const table = useVueTable({
  get data() { return props.data },
  columns: props.columns,
  state: {
    get sorting() { return sorting.value },
    get globalFilter() { return globalFilter.value },
    get pagination() { return pagination.value }
  },
  manualPagination: props.serverSide,
  manualSorting: props.serverSide,
  get rowCount() { return props.serverSide ? props.total : props.data.length },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
  getFilteredRowModel: getFilteredRowModel(),
  onSortingChange: (updater) => {
    sorting.value = typeof updater === 'function' ? updater(sorting.value) : updater
    if (props.serverSide) emit('sort-change', sorting.value)
  },
  onPaginationChange: (updater) => {
    pagination.value = typeof updater === 'function' ? updater(pagination.value) : updater
    if (props.serverSide) emit('page-change', pagination.value)
  }
})

const pageCount = computed(() =>
  props.serverSide
    ? Math.ceil(props.total / pagination.value.pageSize)
    : table.getPageCount()
)
</script>

<template>
  <div class="flex flex-col gap-3">
    <!-- Toolbar slot + optional search -->
    <div class="flex items-center justify-between gap-2">
      <slot name="toolbar" />
      <Input
        v-if="searchable"
        v-model="globalFilter"
        placeholder="Search..."
        class="max-w-xs"
      />
    </div>

    <!-- Table -->
    <div class="overflow-x-auto rounded-lg border border-[var(--color-line)]">
      <Table>
        <TableHeader>
          <TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
          >
            <TableHead
              v-for="header in headerGroup.headers"
              :key="header.id"
              :class="header.column.getCanSort() ? 'cursor-pointer select-none' : ''"
              @click="header.column.getToggleSortingHandler()?.($event)"
            >
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
              <span v-if="header.column.getIsSorted() === 'asc'"> ↑</span>
              <span v-else-if="header.column.getIsSorted() === 'desc'"> ↓</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="loading">
            <TableRow>
              <TableCell :colspan="columns.length" class="py-12 text-center text-[var(--color-text-secondary)]">
                Loading…
              </TableCell>
            </TableRow>
          </template>
          <template v-else-if="table.getRowModel().rows.length === 0">
            <TableRow>
              <TableCell :colspan="columns.length" class="py-12 text-center text-[var(--color-text-secondary)]">
                <slot name="empty">No data</slot>
              </TableCell>
            </TableRow>
          </template>
          <template v-else>
            <TableRow
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              class="hover:bg-[var(--color-surface-muted)] transition-colors"
            >
              <TableCell
                v-for="cell in row.getVisibleCells()"
                :key="cell.id"
              >
                <FlexRender
                  :render="cell.column.columnDef.cell"
                  :props="cell.getContext()"
                />
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
      <span>
        Page {{ pagination.pageIndex + 1 }} of {{ pageCount || 1 }}
      </span>
      <div class="flex gap-1">
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanPreviousPage()"
          @click="table.previousPage()"
        >
          ‹ Prev
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanNextPage()"
          @click="table.nextPage()"
        >
          Next ›
        </Button>
      </div>
    </div>
  </div>
</template>

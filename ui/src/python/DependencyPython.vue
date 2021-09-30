<template>
  <cl-list-layout
      v-loading="loading"
      :table-columns="tableColumns"
      :table-data="tableData"
      :table-total="tableTotal"
      :table-pagination="tablePagination"
      :action-functions="actionFunctions"
      :visible-buttons="['export', 'customize-columns']"
      table-pagination-layout="total, prev, pager, next"
  >
    <template #nav-actions-extra>
      <cl-nav-action-group class="top-bar">
        <el-input
            class="search-query"
            v-model="searchQuery"
            size="small"
            placeholder="Search dependencies"
            prefix-icon="el-icon-search"
            @keyup.enter="onSearch"
        />
        <cl-label-button
            size="small"
            :icon="['fa', 'search']"
            label="Search"
            @click="onSearch"
        />
        <el-checkbox v-model="installed" label="Installed"/>
        <el-pagination
            :current-page="tablePagination.page"
            :page-size="tablePagination.pageSize"
            :total="tableTotal"
            class="pagination"
            layout="total, prev, pager, next"
            @current-change="(page) => tablePagination.page = page"
        />
      </cl-nav-action-group>
    </template>
  </cl-list-layout>
</template>

<script lang="ts">
import {defineComponent, ref, h} from 'vue';
import {useRouter} from 'vue-router';
import {useRequest, ClSwitch, ClNavLink} from 'crawlab-ui';
import DependencyForm from './DependencyForm.vue';
import {ElMessage, ElMessageBox} from 'element-plus';

const endpoint = '/plugin-proxy/dependency/python';

const {
  getList,
  post,
} = useRequest();

const getDefaultForm = () => {
  return {
    type: 'mail',
    enabled: true,
  };
};

export default defineComponent({
  name: 'DependencyPython',
  components: {DependencyForm},
  setup() {
    const tableColumns = [
      {
        key: 'name',
        label: 'Name',
        icon: ['fa', 'font'],
        width: '150',
        value: (row) => h(ClNavLink, {
          label: row.name,
          path: `/notifications/${row._id}`,
        }),
      },
      {
        key: 'version',
        label: 'Version',
        icon: ['fa', 'tag'],
        width: '120',
      },
      {
        key: 'description',
        label: 'Description',
        icon: ['fa', 'comment-alt'],
        width: 'auto',
      },
      {
        key: 'actions',
        label: 'Actions',
        fixed: 'right',
        width: '200',
        buttons: [
          {
            type: 'primary',
            icon: ['fa', 'download'],
            tooltip: 'Install',
            onClick: async (row) => {
              await ElMessageBox.confirm('Are you sure to install?', 'Install');
            },
          },
          // {
          //   type: 'info',
          //   size: 'mini',
          //   icon: ['fa', 'clone'],
          //   tooltip: 'Clone',
          //   onClick: (row) => {
          //     console.log('clone', row);
          //   }
          // },
          {
            type: 'danger',
            size: 'mini',
            icon: ['fa', 'trash-alt'],
            tooltip: 'Delete',
            disabled: (row) => !!row.active,
            onClick: async (row) => {
              // const res = await ElMessageBox.confirm('Are you sure to delete?', 'Delete');
              // if (res) {
              // await deleteById(row._id as string);
              // }
              // await getList();
            },
          },
        ],
        disableTransfer: true,
      },
    ];

    const tableData = ref([]);

    const tablePagination = ref({
      page: 1,
      size: 10,
    });

    const tableTotal = ref(0);

    const installed = ref(false);

    const loading = ref(false);

    const actionFunctions = ref({
      getList: async () => {
        loading.value = true;
        try {
          if (!searchQuery.value) return;
          const res = await getList(`${endpoint}`, {
            ...tablePagination.value,
            query: searchQuery.value,
            installed: installed.value,
          });
          if (!res) {
            tableData.value = [];
            tableTotal.value = 0;
          }
          const {data, total} = res;
          tableData.value = data;
          tableTotal.value = total;
        } catch (e) {
          console.error(e);
        } finally {
          loading.value = false;
        }
      },
      setPagination: (pagination) => {
        tablePagination.value = {...pagination};
      },
    });

    const searchQuery = ref();

    const form = ref(getDefaultForm());

    const dialogVisible = ref(false);

    const navActions = [];

    const onDialogClose = () => {
      dialogVisible.value = false;
      form.value = getDefaultForm();
    };

    const onSearch = async () => {
      await actionFunctions.value.getList();
    };

    return {
      tableColumns,
      tableData,
      tableTotal,
      tablePagination,
      actionFunctions,
      navActions,
      dialogVisible,
      searchQuery,
      form,
      installed,
      loading,
      onDialogClose,
      onSearch,
    };
  },
});
</script>

<style scoped>
.search-query {
  width: 300px;
  margin-right: 10px;
}

.top-bar {
  width: 100%;
  display: flex;
}

.top-bar .pagination {
  width: 100%;
  text-align: right;
}
</style>

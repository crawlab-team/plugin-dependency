<template>
  <cl-list-layout
      v-loading="loading"
      class="dependency-list"
      :table-columns="tableColumns"
      :table-data="tableData"
      :table-total="tableTotal"
      :table-pagination="tablePagination"
      :action-functions="actionFunctions"
      :visible-buttons="['export', 'customize-columns']"
      table-pagination-layout="total, prev, pager, next"
  >
    <template #nav-actions-extra>
      <div class="top-bar">
        <div class="top-bar-left">
          <el-input
              class="search-query"
              v-model="searchQuery"
              size="small"
              placeholder="Search dependencies"
              prefix-icon="el-icon-search"
              clearable
              @keyup.enter="onSearch"
              @clear="onSearchClear"
          />
          <cl-label-button
              class="search-btn"
              size="small"
              :icon="['fa', 'search']"
              label="Search"
              :disabled="!installed ? !searchQuery : false"
              @click="onSearch"
          />
          <el-radio-group
              class="view-mode"
              v-model="viewMode"
              size="small"
              @change="onInstalledChange"
          >
            <el-radio-button label="installed">
              <font-awesome-icon :icon="['fa', 'check']" style="margin-right: 5px"/>
              Installed
            </el-radio-button>
            <el-radio-button label="available">
              <font-awesome-icon :icon="['fab', 'python']" style="margin-right: 5px"/>
              Installable
            </el-radio-button>
          </el-radio-group>
          <cl-fa-icon-button
              class="update-btn"
              size="small"
              type="primary"
              tooltip="Click to update installed dependencies"
              :icon="updateInstalledLoading ? ['fa', 'spinner'] : ['fa', 'sync']"
              :spin="updateInstalledLoading"
              :disabled="updateInstalledLoading"
              @click="onUpdate"
          />
        </div>
        <el-pagination
            :current-page="tablePagination.page"
            :page-size="tablePagination.pageSize"
            :total="tableTotal"
            class="pagination"
            layout="total, prev, pager, next"
            @current-change="(page) => tablePagination.page = page"
        />
      </div>
    </template>
    <template #extra>
      <DependencyPythonInstallForm
          :visible="dialogVisible.install"
          :form="installForm"
          @confirm="onInstall"
          @close="() => onDialogClose('install')"
      />
      <DependencyPythonManageForm
          :visible="dialogVisible.manage"
          :form="manageForm"
          @close="() => onDialogClose('manage')"
      />
    </template>
  </cl-list-layout>
</template>

<script lang="ts">
import {computed, defineComponent, h, onBeforeMount, ref} from 'vue';
import {ClNavLink, ClNodeType, useRequest} from 'crawlab-ui';
import {ElMessageBox} from 'element-plus';
import {useStore} from 'vuex';
import DependencyPythonInstallForm from './DependencyPythonInstallForm.vue';
import DependencyPythonManageForm from './DependencyPythonManageForm.vue';

const endpoint = '/plugin-proxy/dependency/python';

const {
  getList: getList_,
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
  components: {DependencyPythonManageForm, DependencyPythonInstallForm},
  setup() {
    const store = useStore();

    const allNodeListSelectOptions = computed(() => store.getters[`node/allListSelectOptions`]);
    const allNodeDict = computed(() => store.getters[`node/allDict`]);

    const isManageable = (dep) => {
      if (installed.value) return true;
      return !(!dep.result || !dep.result.node_ids);
    };

    const tableColumns = computed(() => {
      return [
        {
          key: 'name',
          label: 'Name',
          icon: ['fa', 'font'],
          width: '200',
          value: (row) => h(ClNavLink, {
            label: row.name,
            path: `https://pypi.org/project/${row.name}`,
            external: true,
          }),
        },
        {
          key: 'version',
          label: 'Latest Version',
          icon: ['fa', 'tag'],
          width: '200',
        },
        {
          key: 'versions',
          label: 'Installed Version',
          icon: ['fa', 'tag'],
          width: '200',
          value: (row) => {
            if (installed.value) {
              if (!row.versions) return;
              return row.versions.join(', ');
            } else {
              if (!row.result || !row.result.versions) return;
              return row.result.versions.join(', ');
            }
          },
        },
        {
          key: 'node_ids',
          label: 'Installed Nodes',
          icon: ['fa', 'server'],
          width: '580',
          value: (row) => {
            const result = (installed.value ? row : row.result) || {};
            const node_ids = result.node_ids || [];
            return node_ids.map(id => {
              const n = allNodeDict.value.get(id);
              if (!n) return;
              return h(ClNodeType, {
                isMaster: n.is_master,
                label: n.name,
              });
            });
          },
        },
        {
          key: 'actions',
          label: 'Actions',
          fixed: 'right',
          width: '200',
          buttons: (row) => {
            if (!isManageable(row)) {
              return [
                {
                  type: 'primary',
                  icon: ['fa', 'download'],
                  tooltip: 'Install',
                  onClick: async (row) => {
                    dialogVisible.value.install = true;
                  },
                },
              ];
            } else {
              return [
                {
                  type: 'warning',
                  icon: ['fa', 'tools'],
                  tooltip: 'Manage',
                  onClick: async (row) => {
                    dialogVisible.value.manage = true;
                  },
                },
              ];
            }
          },
          disableTransfer: true,
        },
      ];
      // return columns.filter(c => {
      //   if (!installed.value) return true;
      //   return c.key !== 'versions';
      //
      // });
    });

    const tableData = ref([]);

    const tablePagination = ref({
      page: 1,
      size: 10,
    });

    const tableTotal = ref(0);

    const viewMode = ref('installed');

    const installed = computed(() => viewMode.value === 'installed');

    const loading = ref(false);

    const updateInstalledLoading = ref(false);

    const getList = async () => {
      loading.value = true;
      try {
        if (!searchQuery.value && !installed.value) {
          tableData.value = [];
          tableTotal.value = 0;
          return;
        }
        const params = {
          ...tablePagination.value,
          query: searchQuery.value,
          installed: installed.value,
        };
        const res = await getList_(`${endpoint}`, params);
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
    };

    const update = async () => {
      updateInstalledLoading.value = true;
      try {
        await post(`${endpoint}/update`);
      } finally {
        setTimeout(() => {
          updateInstalledLoading.value = false;
        }, 5000);
      }
    };

    const actionFunctions = ref({
      getList,
      setPagination: (pagination) => {
        tablePagination.value = {...pagination};
      },
    });

    const searchQuery = ref();

    const form = ref(getDefaultForm());

    const dialogVisible = ref({
      install: false,
      manage: false,
    });

    const navActions = [];

    const onDialogClose = (key) => {
      dialogVisible.value[key] = false;
    };

    const onSearch = async () => {
      await actionFunctions.value.getList();
    };

    const onSearchClear = async () => {
      await actionFunctions.value.getList();
    };

    const onUpdate = async () => {
      await update();
    };

    const onInstalledChange = async () => {
      await actionFunctions.value.getList();
    };

    const onFilterChange = async () => {
      await actionFunctions.value.getList();
    };

    const installForm = ref({});

    const onInstall = async () => {
    };

    const manageForm = ref({});

    onBeforeMount(() => store.dispatch(`node/getAllList`));

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
      viewMode,
      installed,
      loading,
      updateInstalledLoading,
      allNodeListSelectOptions,
      onDialogClose,
      onSearch,
      onSearchClear,
      onUpdate,
      onInstalledChange,
      onFilterChange,
      installForm,
      manageForm,
      onInstall,
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
  align-items: center;
  justify-content: space-between;
  height: 64px;
}

.top-bar >>> .search-btn {
  margin-right: 0;
}

.top-bar >>> .update-btn,
.top-bar >>> .view-mode {
  margin-left: 20px;
}

.top-bar .pagination {
  /*width: 100%;*/
  text-align: right;
}

.dependency-list >>> .node-type {
  margin-right: 10px;
}
</style>

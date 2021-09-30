<template>
  <cl-list-layout
      :table-columns="tableColumns"
      :table-data="tableData"
      :table-total="tableTotal"
      :table-pagination="tablePagination"
      :action-functions="actionFunctions"
      :nav-actions="navActions"
      no-actions
      :visible-buttons="['export', 'customize-columns']"
  >
    <template #extra>
      <cl-create-edit-dialog
          :visible="dialogVisible"
          width="1024px"
          no-batch
          @close="onDialogClose"
      >
      </cl-create-edit-dialog>
    </template>
  </cl-list-layout>
</template>

<script lang="ts">
import {defineComponent, onBeforeMount, ref, h} from 'vue';
import {useRequest, ClNavLink, ClSwitch} from 'crawlab-ui';
import {useRouter} from 'vue-router';

const endpoint = '/plugin-proxy/dependency/settings';

const {
  getList,
  post,
} = useRequest();

export default defineComponent({
  name: 'DependencySettings',
  components: {},
  setup(props, {emit}) {
    const router = useRouter();

    const tableColumns = [
      {
        key: 'name',
        label: 'Name',
        icon: ['fa', 'font'],
        width: '150',
        value: (row) => h(ClNavLink, {
          label: row.name,
          path: `/dependencies/${row.key}`,
        }),
      },
      // {
      //   key: 'enabled',
      //   label: 'Enabled',
      //   icon: ['fa', 'toggle-on'],
      //   width: '120',
      //   value: (row) => h(ClSwitch, {
      //     modelValue: row.enabled,
      //     onChange: async (value) => {
      //       if (!row._id) return;
      //       if (value) {
      //         await post(`${endpoint}/${row._id}/enable`);
      //       } else {
      //         await post(`${endpoint}/${row._id}/disable`);
      //       }
      //     },
      //   }),
      // },
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
            icon: ['fa', 'search'],
            tooltip: 'View',
            onClick: (row) => {
              router.push(`/dependencies/${row.key}`);
            },
          },
          {
            type: 'warning',
            icon: ['fa', 'cog'],
            tooltip: 'Config',
            onClick: (row) => {
              // router.push(`/notifications/${row._id}`);
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

    const actionFunctions = ref({
      getList: async () => {
        const res = await getList(`${endpoint}`, {
          ...tablePagination.value,
        });
        if (!res) {
          tableData.value = [];
          tableTotal.value = 0;
        }
        const {data, total} = res;
        tableData.value = data;
        tableTotal.value = total;
      },
    });

    const dialogVisible = ref(false);

    const onDialogClose = () => {
      // dialogVisible.value = false;
      // form.value = getDefaultForm();
    };

    return {
      tableColumns,
      tableData,
      tableTotal,
      tablePagination,
      actionFunctions,
      dialogVisible,
      onDialogClose,
    };
  },
});
</script>

<style scoped>

</style>

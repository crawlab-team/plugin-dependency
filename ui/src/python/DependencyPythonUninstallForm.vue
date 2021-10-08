<template>
  <cl-dialog
      title="Uninstall"
      :visible="visible"
      width="640px"
      :confirm-loading="loading"
      @confirm="onConfirm"
      @close="onClose"
  >
    <cl-form>
      <cl-form-item :span="4" label="Dependency Name">
        <cl-tag
            v-for="n in names"
            :key="n"
            class="dep-name"
            type="primary"
            :label="n"
            size="small"
        />
      </cl-form-item>
      <cl-form-item :span="4" label="Mode">
        <el-select v-model="mode">
          <el-option value="all" label="All Nodes"/>
          <el-option value="selected-nodes" label="Selected Nodes"/>
        </el-select>
      </cl-form-item>
      <cl-form-item v-if="mode === 'selected-nodes'" :span="4" label="Selected Nodes">
        <el-select v-model="nodeIds" multiple placeholder="Select Nodes">
          <el-option v-for="n in nodes" :key="n.key" :value="n._id" :label="n.name"/>
        </el-select>
      </cl-form-item>
    </cl-form>
  </cl-dialog>
</template>

<script lang="ts">
import {defineComponent, ref} from 'vue';

export default defineComponent({
  name: 'DependencyPythonUninstallForm',
  props: {
    visible: {
      type: Boolean,
    },
    names: {
      type: Array,
      default: () => {
        return [];
      },
    },
    nodes: {
      type: Array,
      default: () => {
        return [];
      }
    },
    loading: {
      type: Boolean,
    },
  },
  emits: [
    'confirm',
    'close',
  ],
  setup(props, {emit}) {
    const mode = ref('all');
    const nodeIds = ref([]);

    const reset = () => {
      mode.value = 'all';
      nodeIds.value = [];
    };

    const onConfirm = () => {
      emit('confirm', {
        mode: mode.value,
        nodeIds: nodeIds.value,
      });
      reset();
    };

    const onClose = () => {
      emit('close');
      reset();
    };

    return {
      nodeIds,
      mode,
      onConfirm,
      onClose,
    };
  },
});
</script>

<style scoped>
.dep-name {
  margin-right: 10px;
}
</style>

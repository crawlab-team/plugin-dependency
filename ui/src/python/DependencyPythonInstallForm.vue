<template>
  <cl-dialog
      title="Install"
      :visible="visible"
  >
    <cl-form :model="form">
      <cl-form-item :span="4" label="Installed Nodes">
        <el-select>
        </el-select>
      </cl-form-item>
    </cl-form>
  </cl-dialog>
</template>

<script lang="ts">
import {defineComponent} from 'vue';

export default defineComponent({
  name: 'DependencyPythonInstallForm',
  props: {
    visible: {
      type: Boolean,
    },
    form: {
      type: Object,
    },
  },
  setup(props, {emit}) {
    return {};
  },
});
</script>

<style scoped>

</style>

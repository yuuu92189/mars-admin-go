"use client";

import { Dialog } from "@/components/ui/overlay";
import { Button } from "@/components/ui/button";
import { Segmented } from "@/components/ui/select";
import { useTheme } from "@/components/providers/theme-provider";

export function AppearanceDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { theme, setTheme, density, setDensity } = useTheme();
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="外观设置"
      description="主题与密度只影响当前浏览器，不写入业务数据。"
      footer={
        <Button variant="primary" onClick={onClose}>
          完成
        </Button>
      }
      width={480}
    >
      <div className="pref-list">
        <div className="pref-row">
          <div>
            <div className="pref-title">主题</div>
            <div className="pref-desc">支持浅色、深色与跟随系统</div>
          </div>
          <Segmented
            label="主题"
            value={theme}
            onChange={(value) => setTheme(value as "light" | "dark" | "system")}
            options={[
              { value: "light", label: "浅色" },
              { value: "dark", label: "深色" },
              { value: "system", label: "系统" },
            ]}
          />
        </div>
        <div className="pref-row">
          <div>
            <div className="pref-title">表格密度</div>
            <div className="pref-desc">控制列表行高与单元格间距</div>
          </div>
          <Segmented
            label="表格密度"
            value={density}
            onChange={(value) => setDensity(value as "comfortable" | "compact")}
            options={[
              { value: "comfortable", label: "舒适" },
              { value: "compact", label: "紧凑" },
            ]}
          />
        </div>
      </div>
    </Dialog>
  );
}

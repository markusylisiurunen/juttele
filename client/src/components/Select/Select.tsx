import classNames from "classnames";
// import { type IconName } from "lucide-react/dynamic";
import React, { useEffect, useState } from "react";
import styles from "./Select.module.css";

type SelectProps = {
  className?: string;
  iconLeft?: React.ReactNode;
  iconRight?: React.ReactNode;
  options: { id: string; label: string }[];
  value: string;
  onChange?: (id: string) => void;
};
const Select: React.FC<SelectProps> = ({
  className,
  iconLeft,
  iconRight,
  options,
  value,
  onChange,
}) => {
  const [selected, setSelected] = useState("");
  useEffect(() => {
    setSelected(value);
  }, [value]);
  function _onChange(e: React.ChangeEvent<HTMLSelectElement>) {
    setSelected(e.target.value);
    onChange?.(e.target.value);
  }
  const label = options.find((option) => option.id === selected)?.label;
  return (
    <div className={classNames(styles.root, className)}>
      <div className={styles.button}>
        {iconLeft}
        <span>{label}</span>
        {iconRight}
      </div>
      <select className={styles.hidden} value={selected} onChange={_onChange}>
        <option value="" disabled>
          {label}
        </option>
        {options.map((option) => (
          <option key={option.id} value={option.id}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
};

export { Select };

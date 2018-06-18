import { h } from 'preact';
import style from './balance.css';

export default function Balance({ name, balance, children }) {
    if (!balance) {
        return (
            <header className={style.balance}></header>
        );
    }
    return (
        <header className={style.balance}>
            <span className={['label', style.label].join(' ')}>{name}</span>
            <span className={style.amount}>
                {balance.available.amount}
                {' '}
                <span className={style.unit}>{balance.available.unit}</span>
            </span>
            {children}
        </header>
    );
}

-- ============================================================================
-- cashflow database schema - full table analysis organized by Odoo module/section
-- Source : PostgreSQL 18.6 - database: cashflow
-- Format : pg_dump 18 --schema-only (full DDL: columns, types, defaults,
--           PK/UNIQUE/CHECK, foreign keys, indexes, triggers, sequences)
-- Sections covered: accounting, inventory, products, sales, purchase, prm, crm,
--           documents, partners, hr, mrp, users, analytics, fleet, livechat, pos, project
-- Tables not belonging to the requested sections are listed last under 'other'.
-- Generated : 2026-09-17
-- Ordering  : all CREATE TABLE + sequences first (in section order), then DEFAULT,
--           PK/UNIQUE/CHECK, foreign keys, indexes and triggers per table.
--           The file remains an importable PostgreSQL schema dump.
-- ============================================================================

--
-- PostgreSQL database dump
--


-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;


-- extension : citext
CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;

-- extension : uuid-ossp
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;

-- function : update_updated_at_column()
CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- ==============================================================================
-- SECTION: accounting Accounting (المحاسبة)
-- ------------------------------------------------------------------------------
-- Tables in this section: 25
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.account_accounts
--   id:                        bigint                     PK NOT NULL
--   code:                      varchar(64)                NOT NULL
--   name:                      jsonb                      NOT NULL
--   type:                      varchar(50)                NOT NULL
--   reconcile:                 boolean                    default=false
--   currency:                  varchar(10)                default='USD'
--   parent_id:                 bigint                     
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   account_stock_variation_id: bigint                     
--   account_stock_expense_id:  bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_accounts
CREATE TABLE public.account_accounts (
    id bigint NOT NULL,
    code character varying(64) NOT NULL,
    name jsonb NOT NULL,
    type character varying(50) NOT NULL,
    reconcile boolean DEFAULT false NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    parent_id bigint,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    account_stock_variation_id bigint,
    account_stock_expense_id bigint
);

-- CREATE SEQUENCE : account_accounts
CREATE SEQUENCE public.account_accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_accounts
ALTER SEQUENCE public.account_accounts_id_seq OWNED BY public.account_accounts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_bank_statement_lines
--   id:                        bigint                     PK NOT NULL
--   statement_id:              bigint                     NOT NULL
--   name:                      varchar(500)               NOT NULL
--   ref:                       varchar(255)               
--   sequence:                  integer                    default=1
--   date:                      date                       default=CURRENT_DATE
--   amount:                    numeric(15,4)              default=0.0
--   amount_currency:           numeric(15,4)              default=0.0
--   currency:                  varchar(10)                default='USD'
--   partner_id:                bigint                     
--   account_id:                bigint                     
--   move_id:                   bigint                     
--   journal_id:                bigint                     
--   checked:                   boolean                    default=false
--   running_balance:           numeric(15,4)              default=0.0
--   amount_residual:           numeric(15,4)              default=0.0
--   reconciled:                boolean                    default=false
--   matching_number:           varchar(64)                
--   internal_index:            varchar(100)               
--   import_batch_id:           varchar(100)               
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_bank_statement_lines
CREATE TABLE public.account_bank_statement_lines (
    id bigint NOT NULL,
    statement_id bigint NOT NULL,
    name character varying(500) NOT NULL,
    ref character varying(255),
    sequence integer DEFAULT 1 NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_currency numeric(15,4) DEFAULT 0.0 NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    partner_id bigint,
    account_id bigint,
    move_id bigint,
    journal_id bigint,
    checked boolean DEFAULT false NOT NULL,
    running_balance numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_residual numeric(15,4) DEFAULT 0.0 NOT NULL,
    reconciled boolean DEFAULT false NOT NULL,
    matching_number character varying(64),
    internal_index character varying(100),
    import_batch_id character varying(100),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_bank_statement_lines
CREATE SEQUENCE public.account_bank_statement_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_bank_statement_lines
ALTER SEQUENCE public.account_bank_statement_lines_id_seq OWNED BY public.account_bank_statement_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_bank_statements
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(100)               NOT NULL
--   journal_id:                bigint                     NOT NULL
--   partner_id:                bigint                     
--   date:                      date                       default=CURRENT_DATE
--   balance_start:             numeric(15,4)              default=0.0
--   balance_end:               numeric(15,4)              default=0.0
--   balance_end_real:          numeric(15,4)              
--   currency:                  varchar(10)                default='USD'
--   state:                     varchar(20)                default='open'
--   is_complete:               boolean                    default=false
--   is_valid:                  boolean                    default=false
--   problem_description:       text                       
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_bank_statements
CREATE TABLE public.account_bank_statements (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    journal_id bigint NOT NULL,
    partner_id bigint,
    date date DEFAULT CURRENT_DATE NOT NULL,
    balance_start numeric(15,4) DEFAULT 0.0 NOT NULL,
    balance_end numeric(15,4) DEFAULT 0.0 NOT NULL,
    balance_end_real numeric(15,4),
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    state character varying(20) DEFAULT 'open'::character varying NOT NULL,
    is_complete boolean DEFAULT false NOT NULL,
    is_valid boolean DEFAULT false NOT NULL,
    problem_description text,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_bank_statements
CREATE SEQUENCE public.account_bank_statements_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_bank_statements
ALTER SEQUENCE public.account_bank_statements_id_seq OWNED BY public.account_bank_statements.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_cash_roundings
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   rounding_method:           varchar(20)                default='HALF-UP'
--   rounding:                  numeric(15,4)              default=0.01
--   strategy:                  varchar(20)                default='add_invoice_line'
--   profit_account_id:         bigint                     
--   loss_account_id:           bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_cash_roundings
CREATE TABLE public.account_cash_roundings (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    rounding_method character varying(20) DEFAULT 'HALF-UP'::character varying NOT NULL,
    rounding numeric(15,4) DEFAULT 0.01 NOT NULL,
    strategy character varying(20) DEFAULT 'add_invoice_line'::character varying NOT NULL,
    profit_account_id bigint,
    loss_account_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_cash_roundings
CREATE SEQUENCE public.account_cash_roundings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_cash_roundings
ALTER SEQUENCE public.account_cash_roundings_id_seq OWNED BY public.account_cash_roundings.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_full_reconciles
--   id:                        bigint                     PK NOT NULL
--   matching_number:           varchar(64)                NOT NULL
--   exchange_move_id:          bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_full_reconciles
CREATE TABLE public.account_full_reconciles (
    id bigint NOT NULL,
    matching_number character varying(64) NOT NULL,
    exchange_move_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_full_reconciles
CREATE SEQUENCE public.account_full_reconciles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_full_reconciles
ALTER SEQUENCE public.account_full_reconciles_id_seq OWNED BY public.account_full_reconciles.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_journals
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   code:                      varchar(20)                NOT NULL
--   type:                      varchar(30)                NOT NULL
--   default_account_id:        bigint                     
--   suspense_account_id:       bigint                     
--   sequence_prefix:           varchar(20)                NOT NULL
--   next_number:               integer                    default=1
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_journals
CREATE TABLE public.account_journals (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    code character varying(20) NOT NULL,
    type character varying(30) NOT NULL,
    default_account_id bigint,
    suspense_account_id bigint,
    sequence_prefix character varying(20) NOT NULL,
    next_number integer DEFAULT 1 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_journals
CREATE SEQUENCE public.account_journals_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_journals
ALTER SEQUENCE public.account_journals_id_seq OWNED BY public.account_journals.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_move_lines
--   id:                        bigint                     PK NOT NULL
--   move_id:                   bigint                     NOT NULL
--   account_id:                bigint                     NOT NULL
--   partner_id:                bigint                     
--   product_id:                bigint                     
--   name:                      varchar(255)               NOT NULL
--   quantity:                  numeric(15,4)              default=1.0
--   price_unit:                numeric(15,4)              default=0.0
--   discount:                  numeric(15,4)              default=0.0
--   debit:                     numeric(15,4)              default=0.0
--   credit:                    numeric(15,4)              default=0.0
--   balance:                   numeric(15,4)              default=0.0
--   tax_ids:                   bigint[]                   default='{}'::bigint[]
--   tax_amount:                numeric(15,4)              default=0.0
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   reconcile:                 boolean                    default=false
--   reconciled:                boolean                    default=false
--   amount_residual:           numeric(15,4)              default=0.0
--   matching_number:           varchar(64)                
--   statement_line_id:         bigint                     
--   display_type:              varchar(20)                
--   cogs_origin_id:            bigint                     
--   is_landed_costs_line:      boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_move_lines
CREATE TABLE public.account_move_lines (
    id bigint NOT NULL,
    move_id bigint NOT NULL,
    account_id bigint NOT NULL,
    partner_id bigint,
    product_id bigint,
    name character varying(255) NOT NULL,
    quantity numeric(15,4) DEFAULT 1.0 NOT NULL,
    price_unit numeric(15,4) DEFAULT 0.0 NOT NULL,
    discount numeric(15,4) DEFAULT 0.0 NOT NULL,
    debit numeric(15,4) DEFAULT 0.0 NOT NULL,
    credit numeric(15,4) DEFAULT 0.0 NOT NULL,
    balance numeric(15,4) DEFAULT 0.0 NOT NULL,
    tax_ids bigint[] DEFAULT '{}'::bigint[],
    tax_amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    reconcile boolean DEFAULT false NOT NULL,
    reconciled boolean DEFAULT false NOT NULL,
    amount_residual numeric(15,4) DEFAULT 0.0 NOT NULL,
    matching_number character varying(64),
    statement_line_id bigint,
    display_type character varying(20),
    cogs_origin_id bigint,
    is_landed_costs_line boolean DEFAULT false NOT NULL
);

-- CREATE SEQUENCE : account_move_lines
CREATE SEQUENCE public.account_move_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_move_lines
ALTER SEQUENCE public.account_move_lines_id_seq OWNED BY public.account_move_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_moves
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(100)               NOT NULL
--   move_type:                 varchar(30)                default='entry'
--   journal_id:                bigint                     NOT NULL
--   partner_id:                bigint                     
--   date:                      date                       default=CURRENT_DATE
--   invoice_date:              date                       
--   invoice_date_due:          date                       
--   payment_term_id:           bigint                     
--   state:                     varchar(20)                default='draft'
--   payment_state:             varchar(20)                default='not_paid'
--   amount_untaxed:            numeric(15,4)              default=0.0
--   amount_tax:                numeric(15,4)              default=0.0
--   amount_total:              numeric(15,4)              default=0.0
--   amount_residual:           numeric(15,4)              default=0.0
--   currency:                  varchar(10)                default='USD'
--   ref:                       varchar(255)               
--   reversed_entry_id:         bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_moves
CREATE TABLE public.account_moves (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    move_type character varying(30) DEFAULT 'entry'::character varying NOT NULL,
    journal_id bigint NOT NULL,
    partner_id bigint,
    date date DEFAULT CURRENT_DATE NOT NULL,
    invoice_date date,
    invoice_date_due date,
    payment_term_id bigint,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    payment_state character varying(20) DEFAULT 'not_paid'::character varying NOT NULL,
    amount_untaxed numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_tax numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_total numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_residual numeric(15,4) DEFAULT 0.0 NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    ref character varying(255),
    reversed_entry_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_moves
CREATE SEQUENCE public.account_moves_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_moves
ALTER SEQUENCE public.account_moves_id_seq OWNED BY public.account_moves.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_partial_reconciles
--   id:                        bigint                     PK NOT NULL
--   debit_move_id:             bigint                     
--   credit_move_id:            bigint                     
--   debit_line_id:             bigint                     NOT NULL
--   credit_line_id:            bigint                     NOT NULL
--   amount:                    numeric(15,4)              default=0.0
--   amount_currency:           numeric(15,4)              default=0.0
--   currency:                  varchar(10)                default='USD'
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_partial_reconciles
CREATE TABLE public.account_partial_reconciles (
    id bigint NOT NULL,
    debit_move_id bigint,
    credit_move_id bigint,
    debit_line_id bigint NOT NULL,
    credit_line_id bigint NOT NULL,
    amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    amount_currency numeric(15,4) DEFAULT 0.0 NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_partial_reconciles
CREATE SEQUENCE public.account_partial_reconciles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_partial_reconciles
ALTER SEQUENCE public.account_partial_reconciles_id_seq OWNED BY public.account_partial_reconciles.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_payment_reconciliations
--   id:                        bigint                     PK NOT NULL
--   payment_id:                bigint                     NOT NULL
--   invoice_id:                bigint                     NOT NULL
--   amount:                    numeric(15,4)              NOT NULL
--   reconciled_at:             timestamptz                default=now()
--   CONSTRAINT:                account_payment_reconciliations_amount_check CHECK ((amount > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_payment_reconciliations
CREATE TABLE public.account_payment_reconciliations (
    id bigint NOT NULL,
    payment_id bigint NOT NULL,
    invoice_id bigint NOT NULL,
    amount numeric(15,4) NOT NULL,
    reconciled_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT account_payment_reconciliations_amount_check CHECK ((amount > (0)::numeric))
);

-- CREATE SEQUENCE : account_payment_reconciliations
CREATE SEQUENCE public.account_payment_reconciliations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_payment_reconciliations
ALTER SEQUENCE public.account_payment_reconciliations_id_seq OWNED BY public.account_payment_reconciliations.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_payment_term_lines
--   id:                        bigint                     PK NOT NULL
--   payment_term_id:           bigint                     NOT NULL
--   value_type:                varchar(20)                default='balance'
--   value_amount:              numeric(15,4)              default=0.0
--   days:                      integer                    default=0
--   day_of_month:              integer                    default=0
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_payment_term_lines
CREATE TABLE public.account_payment_term_lines (
    id bigint NOT NULL,
    payment_term_id bigint NOT NULL,
    value_type character varying(20) DEFAULT 'balance'::character varying NOT NULL,
    value_amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    days integer DEFAULT 0 NOT NULL,
    day_of_month integer DEFAULT 0 NOT NULL
);

-- CREATE SEQUENCE : account_payment_term_lines
CREATE SEQUENCE public.account_payment_term_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_payment_term_lines
ALTER SEQUENCE public.account_payment_term_lines_id_seq OWNED BY public.account_payment_term_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_payment_terms
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   note:                      jsonb                      
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_payment_terms
CREATE TABLE public.account_payment_terms (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    note jsonb,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_payment_terms
CREATE SEQUENCE public.account_payment_terms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_payment_terms
ALTER SEQUENCE public.account_payment_terms_id_seq OWNED BY public.account_payment_terms.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_payments
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(100)               NOT NULL
--   payment_type:              varchar(20)                NOT NULL
--   partner_type:              varchar(20)                default='customer'
--   partner_id:                bigint                     NOT NULL
--   amount:                    numeric(15,4)              NOT NULL
--   currency:                  varchar(10)                default='USD'
--   payment_method:            varchar(50)                default='cash'
--   journal_id:                bigint                     NOT NULL
--   date:                      date                       default=CURRENT_DATE
--   state:                     varchar(20)                default='draft'
--   ref:                       varchar(255)               
--   move_id:                   bigint                     
--   reconciled_amount:         numeric(15,4)              default=0.0
--   residual_amount:           numeric(15,4)              default=0.0
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                account_payments_amount_check CHECK ((amount > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_payments
CREATE TABLE public.account_payments (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    payment_type character varying(20) NOT NULL,
    partner_type character varying(20) DEFAULT 'customer'::character varying NOT NULL,
    partner_id bigint NOT NULL,
    amount numeric(15,4) NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    payment_method character varying(50) DEFAULT 'cash'::character varying NOT NULL,
    journal_id bigint NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    ref character varying(255),
    move_id bigint,
    reconciled_amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    residual_amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT account_payments_amount_check CHECK ((amount > (0)::numeric))
);

-- CREATE SEQUENCE : account_payments
CREATE SEQUENCE public.account_payment_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : account_payments
CREATE SEQUENCE public.account_payments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_payments
ALTER SEQUENCE public.account_payments_id_seq OWNED BY public.account_payments.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_reconcile_model_lines
--   id:                        bigint                     PK NOT NULL
--   reconcile_model_id:        bigint                     NOT NULL
--   amount_type:               varchar(20)                default='fixed'
--   amount:                    varchar(255)               default='0'
--   account_id:                bigint                     
--   label:                     varchar(255)               
--   tax_ids:                   bigint[]                   default='{}'::bigint[]
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_reconcile_model_lines
CREATE TABLE public.account_reconcile_model_lines (
    id bigint NOT NULL,
    reconcile_model_id bigint NOT NULL,
    amount_type character varying(20) DEFAULT 'fixed'::character varying NOT NULL,
    amount character varying(255) DEFAULT '0'::character varying NOT NULL,
    account_id bigint,
    label character varying(255),
    tax_ids bigint[] DEFAULT '{}'::bigint[]
);

-- CREATE SEQUENCE : account_reconcile_model_lines
CREATE SEQUENCE public.account_reconcile_model_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_reconcile_model_lines
ALTER SEQUENCE public.account_reconcile_model_lines_id_seq OWNED BY public.account_reconcile_model_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_reconcile_models
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   sequence:                  integer                    default=10
--   is_auto_reconcile:         boolean                    default=false
--   match_nature:              varchar(20)                default='both'
--   match_amount:              varchar(20)                
--   match_amount_min:          numeric(15,4)              
--   match_amount_max:          numeric(15,4)              
--   match_label:               varchar(20)                
--   match_label_param:         varchar(255)               
--   match_journal_ids:         bigint[]                   default='{}'::bigint[]
--   match_partner_ids:         bigint[]                   default='{}'::bigint[]
--   mapped_partner_id:         bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_reconcile_models
CREATE TABLE public.account_reconcile_models (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    is_auto_reconcile boolean DEFAULT false NOT NULL,
    match_nature character varying(20) DEFAULT 'both'::character varying NOT NULL,
    match_amount character varying(20),
    match_amount_min numeric(15,4),
    match_amount_max numeric(15,4),
    match_label character varying(20),
    match_label_param character varying(255),
    match_journal_ids bigint[] DEFAULT '{}'::bigint[],
    match_partner_ids bigint[] DEFAULT '{}'::bigint[],
    mapped_partner_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_reconcile_models
CREATE SEQUENCE public.account_reconcile_models_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_reconcile_models
ALTER SEQUENCE public.account_reconcile_models_id_seq OWNED BY public.account_reconcile_models.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_taxes
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   type:                      varchar(20)                default='percent'
--   type_tax_use:              varchar(20)                default='sale'
--   amount:                    numeric(15,4)              default=0.0
--   account_id:                bigint                     
--   refund_account_id:         bigint                     
--   price_include:             boolean                    default=false
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_taxes
CREATE TABLE public.account_taxes (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    type character varying(20) DEFAULT 'percent'::character varying NOT NULL,
    type_tax_use character varying(20) DEFAULT 'sale'::character varying NOT NULL,
    amount numeric(15,4) DEFAULT 0.0 NOT NULL,
    account_id bigint,
    refund_account_id bigint,
    price_include boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : account_taxes
CREATE SEQUENCE public.account_taxes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_taxes
ALTER SEQUENCE public.account_taxes_id_seq OWNED BY public.account_taxes.id;

-- ----------------------------------------------------------------------
-- TABLE: public.accounting_periods
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   date_from:                 date                       NOT NULL
--   date_to:                   date                       NOT NULL
--   state:                     varchar(16)                default='open'
--   journal_id:                bigint                     
--   account_move_id:           bigint                     
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.accounting_periods
CREATE TABLE public.accounting_periods (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    date_from date NOT NULL,
    date_to date NOT NULL,
    state character varying(16) DEFAULT 'open'::character varying NOT NULL,
    journal_id bigint,
    account_move_id bigint,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint
);

-- CREATE SEQUENCE : accounting_periods
CREATE SEQUENCE public.accounting_periods_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : accounting_periods
ALTER SEQUENCE public.accounting_periods_id_seq OWNED BY public.accounting_periods.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_provider_configs
--   id:                        bigint                     PK NOT NULL
--   provider_id:               bigint                     NOT NULL
--   key:                       varchar(128)               NOT NULL
--   value:                     text                       NOT NULL
--   is_secret:                 boolean                    default=false
--   environment:               varchar(16)                default='test'
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_provider_configs
CREATE TABLE public.payment_provider_configs (
    id bigint NOT NULL,
    provider_id bigint NOT NULL,
    key character varying(128) NOT NULL,
    value text NOT NULL,
    is_secret boolean DEFAULT false NOT NULL,
    environment character varying(16) DEFAULT 'test'::character varying NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : payment_provider_configs
CREATE SEQUENCE public.payment_provider_configs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_provider_configs
ALTER SEQUENCE public.payment_provider_configs_id_seq OWNED BY public.payment_provider_configs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_providers
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   code:                      varchar(50)                NOT NULL
--   state:                     varchar(20)                default='disabled'
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   module_state:              varchar(32)                default='installed'
--   inline_form:               boolean                    default=false
--   support_refund:            varchar(16)                default='none'
--   support_tokenize:          boolean                    default=false
--   support_authorize:         boolean                    default=false
--   webhook_secret:            text                       
--   allow_tokenize:            boolean                    default=false
--   capture_manually:          boolean                    default=false
--   journal_id:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_providers
CREATE TABLE public.payment_providers (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    code character varying(50) NOT NULL,
    state character varying(20) DEFAULT 'disabled'::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    module_state character varying(32) DEFAULT 'installed'::character varying,
    inline_form boolean DEFAULT false,
    support_refund character varying(16) DEFAULT 'none'::character varying,
    support_tokenize boolean DEFAULT false,
    support_authorize boolean DEFAULT false,
    webhook_secret text,
    allow_tokenize boolean DEFAULT false,
    capture_manually boolean DEFAULT false,
    journal_id bigint
);

-- CREATE SEQUENCE : payment_providers
CREATE SEQUENCE public.payment_providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_providers
ALTER SEQUENCE public.payment_providers_id_seq OWNED BY public.payment_providers.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_refunds
--   id:                        bigint                     PK NOT NULL
--   original_transaction_id:   bigint                     NOT NULL
--   refund_transaction_id:     bigint                     
--   amount:                    numeric(15,4)              NOT NULL
--   currency:                  varchar(3)                 NOT NULL
--   reason:                    varchar(64)                NOT NULL
--   provider_reference:        varchar(256)               
--   state:                     varchar(32)                default='pending'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_refunds
CREATE TABLE public.payment_refunds (
    id bigint NOT NULL,
    original_transaction_id bigint NOT NULL,
    refund_transaction_id bigint,
    amount numeric(15,4) NOT NULL,
    currency character varying(3) NOT NULL,
    reason character varying(64) NOT NULL,
    provider_reference character varying(256),
    state character varying(32) DEFAULT 'pending'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : payment_refunds
CREATE SEQUENCE public.payment_refunds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_refunds
ALTER SEQUENCE public.payment_refunds_id_seq OWNED BY public.payment_refunds.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_tokens
--   id:                        bigint                     PK NOT NULL
--   provider_id:               bigint                     NOT NULL
--   partner_id:                bigint                     NOT NULL
--   provider_ref:              varchar(256)               NOT NULL
--   display_name:              varchar(128)               NOT NULL
--   payment_details:           varchar(64)                
--   active:                    boolean                    default=true
--   verified:                  boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_tokens
CREATE TABLE public.payment_tokens (
    id bigint NOT NULL,
    provider_id bigint NOT NULL,
    partner_id bigint NOT NULL,
    provider_ref character varying(256) NOT NULL,
    display_name character varying(128) NOT NULL,
    payment_details character varying(64),
    active boolean DEFAULT true NOT NULL,
    verified boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : payment_tokens
CREATE SEQUENCE public.payment_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_tokens
ALTER SEQUENCE public.payment_tokens_id_seq OWNED BY public.payment_tokens.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_transactions
--   id:                        bigint                     PK NOT NULL
--   reference:                 varchar(255)               NOT NULL
--   amount:                    numeric(15,4)              NOT NULL
--   currency:                  varchar(10)                NOT NULL
--   provider_id:               bigint                     NOT NULL
--   partner_id:                bigint                     NOT NULL
--   state:                     varchar(20)                default='draft'
--   provider_reference:        varchar(255)               
--   sale_order_id:             bigint                     
--   invoice_id:                bigint                     
--   payment_id:                bigint                     
--   idempotency_key:           varchar(255)               
--   return_url:                text                       
--   webhook_received:          boolean                    default=false
--   last_error:                text                       
--   metadata:                  jsonb                      
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_transactions
CREATE TABLE public.payment_transactions (
    id bigint NOT NULL,
    reference character varying(255) NOT NULL,
    amount numeric(15,4) NOT NULL,
    currency character varying(10) NOT NULL,
    provider_id bigint NOT NULL,
    partner_id bigint NOT NULL,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    provider_reference character varying(255),
    sale_order_id bigint,
    invoice_id bigint,
    payment_id bigint,
    idempotency_key character varying(255),
    return_url text,
    webhook_received boolean DEFAULT false NOT NULL,
    last_error text,
    metadata jsonb,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : payment_transactions
CREATE SEQUENCE public.payment_transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_transactions
ALTER SEQUENCE public.payment_transactions_id_seq OWNED BY public.payment_transactions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.payment_webhook_logs
--   id:                        bigint                     PK NOT NULL
--   provider_code:             varchar(64)                NOT NULL
--   event_type:                varchar(128)               NOT NULL
--   payload:                   jsonb                      NOT NULL
--   processed:                 boolean                    default=false
--   process_error:             text                       
--   idempotency_key:           varchar(256)               NOT NULL
--   received_at:               timestamptz                default=now()
--   processed_at:              timestamptz                
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.payment_webhook_logs
CREATE TABLE public.payment_webhook_logs (
    id bigint NOT NULL,
    provider_code character varying(64) NOT NULL,
    event_type character varying(128) NOT NULL,
    payload jsonb NOT NULL,
    processed boolean DEFAULT false NOT NULL,
    process_error text,
    idempotency_key character varying(256) NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    processed_at timestamp with time zone
);

-- CREATE SEQUENCE : payment_webhook_logs
CREATE SEQUENCE public.payment_webhook_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : payment_webhook_logs
ALTER SEQUENCE public.payment_webhook_logs_id_seq OWNED BY public.payment_webhook_logs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_currencies
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(10)                NOT NULL
--   full_name:                 varchar(100)               NOT NULL
--   symbol:                    varchar(10)                NOT NULL
--   decimal_places:            smallint                   default=2
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                res_currencies_decimal_places_check CHECK (((decimal_places >= 0) AND (decimal_places <= 10))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_currencies
CREATE TABLE public.res_currencies (
    id bigint NOT NULL,
    name character varying(10) NOT NULL,
    full_name character varying(100) NOT NULL,
    symbol character varying(10) NOT NULL,
    decimal_places smallint DEFAULT 2 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT res_currencies_decimal_places_check CHECK (((decimal_places >= 0) AND (decimal_places <= 10)))
);

-- CREATE SEQUENCE : res_currencies
CREATE SEQUENCE public.res_currencies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_currencies
ALTER SEQUENCE public.res_currencies_id_seq OWNED BY public.res_currencies.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_currency_rates
--   id:                        bigint                     PK NOT NULL
--   currency_id:               bigint                     NOT NULL
--   rate:                      numeric(20,10)             default=1.0
--   date:                      date                       NOT NULL
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                res_currency_rates_rate_check CHECK ((rate > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_currency_rates
CREATE TABLE public.res_currency_rates (
    id bigint NOT NULL,
    currency_id bigint NOT NULL,
    rate numeric(20,10) DEFAULT 1.0 NOT NULL,
    date date NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT res_currency_rates_rate_check CHECK ((rate > (0)::numeric))
);

-- CREATE SEQUENCE : res_currency_rates
CREATE SEQUENCE public.res_currency_rates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_currency_rates
ALTER SEQUENCE public.res_currency_rates_id_seq OWNED BY public.res_currency_rates.id;

-- ==============================================================================
-- SECTION: inventory  Inventory / Stock (المخزون / المعدات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 28
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_carrier
--   id:                        integer                    PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   active:                    boolean                    default=true
--   sequence:                  integer                    default=10
--   delivery_type:             varchar(50)                NOT NULL
--   integration_level:         varchar(50)                default='rate'
--   invoice_policy:            varchar(50)                default='estimated'
--   product_id:                bigint                     NOT NULL
--   fixed_price:               numeric(19,4)              default=0
--   margin:                    numeric(19,4)              default=0
--   fixed_margin:              numeric(19,4)              default=0
--   free_over:                 boolean                    default=false
--   amount:                    numeric(19,4)              default=0
--   max_weight:                numeric(19,4)              
--   max_volume:                numeric(19,4)              
--   company_id:                bigint                     
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_carrier
CREATE TABLE public.delivery_carrier (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    active boolean DEFAULT true,
    sequence integer DEFAULT 10,
    delivery_type character varying(50) NOT NULL,
    integration_level character varying(50) DEFAULT 'rate'::character varying,
    invoice_policy character varying(50) DEFAULT 'estimated'::character varying,
    product_id bigint NOT NULL,
    fixed_price numeric(19,4) DEFAULT 0,
    margin numeric(19,4) DEFAULT 0,
    fixed_margin numeric(19,4) DEFAULT 0,
    free_over boolean DEFAULT false,
    amount numeric(19,4) DEFAULT 0,
    max_weight numeric(19,4),
    max_volume numeric(19,4),
    company_id bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

-- CREATE SEQUENCE : delivery_carrier
CREATE SEQUENCE public.delivery_carrier_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : delivery_carrier
ALTER SEQUENCE public.delivery_carrier_id_seq OWNED BY public.delivery_carrier.id;

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_carrier_country_rel
--   carrier_id:                integer                    PK NOT NULL
--   country_id:                bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_carrier_country_rel
CREATE TABLE public.delivery_carrier_country_rel (
    carrier_id integer NOT NULL,
    country_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_carrier_state_rel
--   carrier_id:                integer                    PK NOT NULL
--   state_id:                  bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_carrier_state_rel
CREATE TABLE public.delivery_carrier_state_rel (
    carrier_id integer NOT NULL,
    state_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_carrier_zip_prefix_rel
--   carrier_id:                integer                    PK NOT NULL
--   zip_prefix_id:             integer                    PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_carrier_zip_prefix_rel
CREATE TABLE public.delivery_carrier_zip_prefix_rel (
    carrier_id integer NOT NULL,
    zip_prefix_id integer NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_price_rule
--   id:                        integer                    PK NOT NULL
--   carrier_id:                integer                    NOT NULL
--   sequence:                  integer                    default=10
--   variable:                  varchar(50)                NOT NULL
--   operator:                  varchar(10)                NOT NULL
--   max_value:                 numeric(19,4)              NOT NULL
--   list_base_price:           numeric(19,4)              default=0
--   list_price:                numeric(19,4)              default=0
--   variable_factor:           varchar(50)                default='weight'
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_price_rule
CREATE TABLE public.delivery_price_rule (
    id integer NOT NULL,
    carrier_id integer NOT NULL,
    sequence integer DEFAULT 10,
    variable character varying(50) NOT NULL,
    operator character varying(10) NOT NULL,
    max_value numeric(19,4) NOT NULL,
    list_base_price numeric(19,4) DEFAULT 0,
    list_price numeric(19,4) DEFAULT 0,
    variable_factor character varying(50) DEFAULT 'weight'::character varying
);

-- CREATE SEQUENCE : delivery_price_rule
CREATE SEQUENCE public.delivery_price_rule_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : delivery_price_rule
ALTER SEQUENCE public.delivery_price_rule_id_seq OWNED BY public.delivery_price_rule.id;

-- ----------------------------------------------------------------------
-- TABLE: public.delivery_zip_prefix
--   id:                        integer                    PK NOT NULL
--   name:                      varchar(100)               NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.delivery_zip_prefix
CREATE TABLE public.delivery_zip_prefix (
    id integer NOT NULL,
    name character varying(100) NOT NULL
);

-- CREATE SEQUENCE : delivery_zip_prefix
CREATE SEQUENCE public.delivery_zip_prefix_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : delivery_zip_prefix
ALTER SEQUENCE public.delivery_zip_prefix_id_seq OWNED BY public.delivery_zip_prefix.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_account_config
--   id:                        bigint                     PK NOT NULL
--   company_id:                bigint                     NOT NULL
--   product_category_id:       bigint                     
--   stock_valuation_account_id: bigint                     NOT NULL
--   stock_input_account_id:    bigint                     NOT NULL
--   stock_output_account_id:   bigint                     NOT NULL
--   stock_journal_id:          bigint                     NOT NULL
--   price_diff_account_id:     bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_account_config
CREATE TABLE public.stock_account_config (
    id bigint NOT NULL,
    company_id bigint NOT NULL,
    product_category_id bigint,
    stock_valuation_account_id bigint NOT NULL,
    stock_input_account_id bigint NOT NULL,
    stock_output_account_id bigint NOT NULL,
    stock_journal_id bigint NOT NULL,
    price_diff_account_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_account_config
CREATE SEQUENCE public.stock_account_config_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_account_config
ALTER SEQUENCE public.stock_account_config_id_seq OWNED BY public.stock_account_config.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_barcode_nomenclatures
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_barcode_nomenclatures
CREATE TABLE public.stock_barcode_nomenclatures (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : stock_barcode_nomenclatures
CREATE SEQUENCE public.stock_barcode_nomenclatures_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_barcode_nomenclatures
ALTER SEQUENCE public.stock_barcode_nomenclatures_id_seq OWNED BY public.stock_barcode_nomenclatures.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_barcode_rules
--   id:                        bigint                     PK NOT NULL
--   nomenclature_id:           bigint                     NOT NULL
--   name:                      varchar(128)               NOT NULL
--   sequence:                  integer                    default=10
--   encoding:                  varchar(32)                NOT NULL
--   type:                      varchar(32)                NOT NULL
--   pattern:                   text                       NOT NULL
--   gs1_content_type:          varchar(64)                
--   associated:                boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_barcode_rules
CREATE TABLE public.stock_barcode_rules (
    id bigint NOT NULL,
    nomenclature_id bigint NOT NULL,
    name character varying(128) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    encoding character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    pattern text NOT NULL,
    gs1_content_type character varying(64),
    associated boolean DEFAULT false NOT NULL
);

-- CREATE SEQUENCE : stock_barcode_rules
CREATE SEQUENCE public.stock_barcode_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_barcode_rules
ALTER SEQUENCE public.stock_barcode_rules_id_seq OWNED BY public.stock_barcode_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_landed_cost_lines
--   id:                        bigint                     PK NOT NULL
--   landed_cost_id:            bigint                     NOT NULL
--   name:                      varchar(512)               NOT NULL
--   product_id:                bigint                     
--   account_id:                bigint                     
--   price_unit:                numeric(15,4)              default=0.0
--   split_method:              varchar(24)                default='equal'
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_landed_cost_lines
CREATE TABLE public.stock_landed_cost_lines (
    id bigint NOT NULL,
    landed_cost_id bigint NOT NULL,
    name character varying(512) NOT NULL,
    product_id bigint,
    account_id bigint,
    price_unit numeric(15,4) DEFAULT 0.0 NOT NULL,
    split_method character varying(24) DEFAULT 'equal'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_landed_cost_lines
CREATE SEQUENCE public.stock_landed_cost_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_landed_cost_lines
ALTER SEQUENCE public.stock_landed_cost_lines_id_seq OWNED BY public.stock_landed_cost_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_landed_costs
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   date:                      date                       NOT NULL
--   state:                     varchar(16)                default='draft'
--   picking_ids:               bigint[]                   default='{}'::bigint[]
--   amount_total:              numeric(15,4)              default=0.0
--   description:               text                       
--   account_move_id:           bigint                     
--   journal_id:                bigint                     
--   vendor_bill_id:            bigint                     
--   company_id:                bigint                     default=1
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_landed_costs
CREATE TABLE public.stock_landed_costs (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    date date NOT NULL,
    state character varying(16) DEFAULT 'draft'::character varying NOT NULL,
    picking_ids bigint[] DEFAULT '{}'::bigint[] NOT NULL,
    amount_total numeric(15,4) DEFAULT 0.0 NOT NULL,
    description text,
    account_move_id bigint,
    journal_id bigint,
    vendor_bill_id bigint,
    company_id bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint
);

-- CREATE SEQUENCE : stock_landed_costs
CREATE SEQUENCE public.stock_landed_cost_sequence
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : stock_landed_costs
CREATE SEQUENCE public.stock_landed_costs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_landed_costs
ALTER SEQUENCE public.stock_landed_costs_id_seq OWNED BY public.stock_landed_costs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_locations
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   complete_name:             jsonb                      NOT NULL
--   usage:                     varchar(32)                default='internal'
--   parent_id:                 bigint                     
--   scrap_location:            boolean                    default=false
--   return_location:           boolean                    default=false
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   valuation_account_id:      bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_locations
CREATE TABLE public.stock_locations (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    complete_name jsonb NOT NULL,
    usage character varying(32) DEFAULT 'internal'::character varying NOT NULL,
    parent_id bigint,
    scrap_location boolean DEFAULT false NOT NULL,
    return_location boolean DEFAULT false NOT NULL,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    valuation_account_id bigint
);

-- CREATE SEQUENCE : stock_locations
CREATE SEQUENCE public.stock_locations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_locations
ALTER SEQUENCE public.stock_locations_id_seq OWNED BY public.stock_locations.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_lots
--   id:                        bigint                     PK NOT NULL
--   product_id:                bigint                     NOT NULL
--   name:                      varchar(128)               NOT NULL
--   tracking_mode:             varchar(16)                default='lot'
--   company_id:                bigint                     
--   expiration_at:             timestamptz                
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                stock_lots_tracking_mode CHECK (((tracking_mode)::text = ANY ((ARRAY['lot'::varchar, 'serial'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_lots
CREATE TABLE public.stock_lots (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    name character varying(128) NOT NULL,
    tracking_mode character varying(16) DEFAULT 'lot'::character varying NOT NULL,
    company_id bigint,
    expiration_at timestamp with time zone,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT stock_lots_tracking_mode CHECK (((tracking_mode)::text = ANY ((ARRAY['lot'::character varying, 'serial'::character varying])::text[])))
);

-- CREATE SEQUENCE : stock_lots
CREATE SEQUENCE public.stock_lots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_lots
ALTER SEQUENCE public.stock_lots_id_seq OWNED BY public.stock_lots.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_move_lines
--   id:                        bigint                     PK NOT NULL
--   move_id:                   bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   product_uom:               bigint                     
--   lot_id:                    bigint                     
--   package_id:                bigint                     
--   owner_id:                  bigint                     
--   location_id:               bigint                     NOT NULL
--   location_dest_id:          bigint                     NOT NULL
--   reserved_quantity:         numeric(15,4)              default=0.0000
--   quantity_done:             numeric(15,4)              default=0.0000
--   date:                      timestamptz                default=now()
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   tracking_mode:             varchar(16)                default='none'
--   result_package_id:         bigint                     
--   CONSTRAINT:                stock_move_lines_nonnegative CHECK (((reserved_quantity >= (0)::numeric) AND (quantity_done >= (0)::numeric))) 
--   CONSTRAINT:                stock_move_lines_tracking_check CHECK (((tracking_mode)::text = ANY ((ARRAY['none'::varchar, 'lot'::varchar, 'serial'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_move_lines
CREATE TABLE public.stock_move_lines (
    id bigint NOT NULL,
    move_id bigint NOT NULL,
    product_id bigint NOT NULL,
    product_uom bigint,
    lot_id bigint,
    package_id bigint,
    owner_id bigint,
    location_id bigint NOT NULL,
    location_dest_id bigint NOT NULL,
    reserved_quantity numeric(15,4) DEFAULT 0.0000 NOT NULL,
    quantity_done numeric(15,4) DEFAULT 0.0000 NOT NULL,
    date timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    tracking_mode character varying(16) DEFAULT 'none'::character varying NOT NULL,
    result_package_id bigint,
    CONSTRAINT stock_move_lines_nonnegative CHECK (((reserved_quantity >= (0)::numeric) AND (quantity_done >= (0)::numeric))),
    CONSTRAINT stock_move_lines_tracking_check CHECK (((tracking_mode)::text = ANY ((ARRAY['none'::character varying, 'lot'::character varying, 'serial'::character varying])::text[])))
);

-- CREATE SEQUENCE : stock_move_lines
CREATE SEQUENCE public.stock_move_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_move_lines
ALTER SEQUENCE public.stock_move_lines_id_seq OWNED BY public.stock_move_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_moves
--   id:                        bigint                     PK NOT NULL
--   picking_id:                bigint                     
--   sequence:                  integer                    default=10
--   name:                      varchar(255)               NOT NULL
--   product_id:                bigint                     NOT NULL
--   product_uom:               bigint                     
--   product_qty:               numeric(15,4)              default=1.0000
--   quantity_done:             numeric(15,4)              default=0.0000
--   location_id:               bigint                     NOT NULL
--   location_dest_id:          bigint                     NOT NULL
--   state:                     varchar(20)                default='draft'
--   sale_line_id:              bigint                     
--   purchase_line_id:          bigint                     
--   date:                      timestamptz                default=now()
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   value:                     numeric(15,4)              default=0.0
--   value_manual:              numeric(15,4)              
--   standard_price:            numeric(15,4)              default=0.0
--   is_in:                     boolean                    default=false
--   is_out:                    boolean                    default=false
--   is_dropship:               boolean                    default=false
--   remaining_qty:             numeric(15,4)              default=0.0
--   remaining_value:           numeric(15,4)              default=0.0
--   account_move_id:           bigint                     
--   procurement_group_id:      bigint                     
--   production_id:             bigint                     
--   production_finished_id:    bigint                     
--   reserved_quantity:         numeric(15,4)              default=0.0000
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_moves
CREATE TABLE public.stock_moves (
    id bigint NOT NULL,
    picking_id bigint,
    sequence integer DEFAULT 10 NOT NULL,
    name character varying(255) NOT NULL,
    product_id bigint NOT NULL,
    product_uom bigint,
    product_qty numeric(15,4) DEFAULT 1.0000 NOT NULL,
    quantity_done numeric(15,4) DEFAULT 0.0000 NOT NULL,
    location_id bigint NOT NULL,
    location_dest_id bigint NOT NULL,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    sale_line_id bigint,
    purchase_line_id bigint,
    date timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    value numeric(15,4) DEFAULT 0.0 NOT NULL,
    value_manual numeric(15,4),
    standard_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    is_in boolean DEFAULT false NOT NULL,
    is_out boolean DEFAULT false NOT NULL,
    is_dropship boolean DEFAULT false NOT NULL,
    remaining_qty numeric(15,4) DEFAULT 0.0 NOT NULL,
    remaining_value numeric(15,4) DEFAULT 0.0 NOT NULL,
    account_move_id bigint,
    procurement_group_id bigint,
    production_id bigint,
    production_finished_id bigint,
    reserved_quantity numeric(15,4) DEFAULT 0.0000 NOT NULL
);

-- CREATE SEQUENCE : stock_moves
CREATE SEQUENCE public.stock_moves_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_moves
ALTER SEQUENCE public.stock_moves_id_seq OWNED BY public.stock_moves.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_orderpoints
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   product_id:                bigint                     NOT NULL
--   warehouse_id:              bigint                     
--   location_id:               bigint                     NOT NULL
--   vendor_id:                 bigint                     
--   min_qty:                   numeric(15,4)              default=0.0
--   max_qty:                   numeric(15,4)              default=0.0
--   qty_multiple:              numeric(15,4)              default=1.0
--   lead_days:                 integer                    default=0
--   source:                    varchar(16)                default='buy'
--   trigger:                   varchar(16)                default='manual'
--   snoozed_until:             timestamptz                
--   qty_on_hand:               numeric(15,4)              default=0.0
--   qty_forecast:              numeric(15,4)              default=0.0
--   qty_to_order:              numeric(15,4)              default=0.0
--   qty_to_order_manual:       numeric(15,4)              default=0.0
--   deadline_date:             date                       
--   active:                    boolean                    default=true
--   company_id:                bigint                     default=1
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_orderpoints
CREATE TABLE public.stock_orderpoints (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    product_id bigint NOT NULL,
    warehouse_id bigint,
    location_id bigint NOT NULL,
    vendor_id bigint,
    min_qty numeric(15,4) DEFAULT 0.0 NOT NULL,
    max_qty numeric(15,4) DEFAULT 0.0 NOT NULL,
    qty_multiple numeric(15,4) DEFAULT 1.0 NOT NULL,
    lead_days integer DEFAULT 0 NOT NULL,
    source character varying(16) DEFAULT 'buy'::character varying NOT NULL,
    trigger character varying(16) DEFAULT 'manual'::character varying NOT NULL,
    snoozed_until timestamp with time zone,
    qty_on_hand numeric(15,4) DEFAULT 0.0 NOT NULL,
    qty_forecast numeric(15,4) DEFAULT 0.0 NOT NULL,
    qty_to_order numeric(15,4) DEFAULT 0.0 NOT NULL,
    qty_to_order_manual numeric(15,4) DEFAULT 0.0 NOT NULL,
    deadline_date date,
    active boolean DEFAULT true NOT NULL,
    company_id bigint DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_orderpoints
CREATE SEQUENCE public.stock_orderpoint_sequence
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : stock_orderpoints
CREATE SEQUENCE public.stock_orderpoints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_orderpoints
ALTER SEQUENCE public.stock_orderpoints_id_seq OWNED BY public.stock_orderpoints.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_package_types
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   height:                    numeric(10,2)              default=0
--   width:                     numeric(10,2)              default=0
--   length:                    numeric(10,2)              default=0
--   max_weight:                numeric(10,2)              default=0
--   barcode:                   varchar(128)               
--   sequence:                  integer                    default=10
--   company_id:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_package_types
CREATE TABLE public.stock_package_types (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    height numeric(10,2) DEFAULT 0 NOT NULL,
    width numeric(10,2) DEFAULT 0 NOT NULL,
    length numeric(10,2) DEFAULT 0 NOT NULL,
    max_weight numeric(10,2) DEFAULT 0 NOT NULL,
    barcode character varying(128),
    sequence integer DEFAULT 10 NOT NULL,
    company_id bigint
);

-- CREATE SEQUENCE : stock_package_types
CREATE SEQUENCE public.stock_package_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_package_types
ALTER SEQUENCE public.stock_package_types_id_seq OWNED BY public.stock_package_types.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_packages
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   package_type_id:           bigint                     
--   location_id:               bigint                     NOT NULL
--   company_id:                bigint                     NOT NULL
--   weight:                    numeric(10,4)              default=0
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_packages
CREATE TABLE public.stock_packages (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    package_type_id bigint,
    location_id bigint NOT NULL,
    company_id bigint NOT NULL,
    weight numeric(10,4) DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_packages
CREATE SEQUENCE public.stock_packages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_packages
ALTER SEQUENCE public.stock_packages_id_seq OWNED BY public.stock_packages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_pickings
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   picking_type:              varchar(20)                NOT NULL
--   state:                     varchar(20)                default='draft'
--   partner_id:                bigint                     
--   location_id:               bigint                     NOT NULL
--   location_dest_id:          bigint                     NOT NULL
--   scheduled_date:            timestamptz                default=now()
--   date_done:                 timestamptz                
--   origin:                    varchar(128)               
--   source_order_id:           bigint                     
--   company_id:                bigint                     
--   note:                      text                       
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   procurement_group_id:      bigint                     
--   carrier_id:                integer                    
--   carrier_tracking_ref:      varchar(255)               
--   weight:                    numeric(19,4)              default=0
--   shipping_weight:           numeric(19,4)              default=0
--   number_of_packages:        integer                    default=0
--   backorder_of_id:           bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_pickings
CREATE TABLE public.stock_pickings (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    picking_type character varying(20) NOT NULL,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    partner_id bigint,
    location_id bigint NOT NULL,
    location_dest_id bigint NOT NULL,
    scheduled_date timestamp with time zone DEFAULT now() NOT NULL,
    date_done timestamp with time zone,
    origin character varying(128),
    source_order_id bigint,
    company_id bigint,
    note text,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    procurement_group_id bigint,
    carrier_id integer,
    carrier_tracking_ref character varying(255),
    weight numeric(19,4) DEFAULT 0,
    shipping_weight numeric(19,4) DEFAULT 0,
    number_of_packages integer DEFAULT 0,
    backorder_of_id bigint
);

-- CREATE SEQUENCE : stock_pickings
CREATE SEQUENCE public.stock_picking_in_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : stock_pickings
CREATE SEQUENCE public.stock_picking_int_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : stock_pickings
CREATE SEQUENCE public.stock_picking_out_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : stock_pickings
CREATE SEQUENCE public.stock_pickings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_pickings
ALTER SEQUENCE public.stock_pickings_id_seq OWNED BY public.stock_pickings.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_procurement_groups
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_procurement_groups
CREATE TABLE public.stock_procurement_groups (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_procurement_groups
CREATE SEQUENCE public.stock_procurement_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_procurement_groups
ALTER SEQUENCE public.stock_procurement_groups_id_seq OWNED BY public.stock_procurement_groups.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_putaway_rules
--   id:                        bigint                     PK NOT NULL
--   product_id:                bigint                     
--   category_id:               bigint                     
--   location_in_id:            bigint                     NOT NULL
--   location_out_id:           bigint                     NOT NULL
--   storage_category_id:       bigint                     
--   sequence:                  integer                    default=10
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_putaway_rules
CREATE TABLE public.stock_putaway_rules (
    id bigint NOT NULL,
    product_id bigint,
    category_id bigint,
    location_in_id bigint NOT NULL,
    location_out_id bigint NOT NULL,
    storage_category_id bigint,
    sequence integer DEFAULT 10 NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : stock_putaway_rules
CREATE SEQUENCE public.stock_putaway_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_putaway_rules
ALTER SEQUENCE public.stock_putaway_rules_id_seq OWNED BY public.stock_putaway_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_quants
--   id:                        bigint                     PK NOT NULL
--   product_id:                bigint                     NOT NULL
--   location_id:               bigint                     NOT NULL
--   quantity:                  numeric(15,4)              default=0.0000
--   reserved_quantity:         numeric(15,4)              default=0.0000
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   package_id:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_quants
CREATE TABLE public.stock_quants (
    id bigint NOT NULL,
    product_id bigint NOT NULL,
    location_id bigint NOT NULL,
    quantity numeric(15,4) DEFAULT 0.0000 NOT NULL,
    reserved_quantity numeric(15,4) DEFAULT 0.0000 NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    package_id bigint
);

-- CREATE SEQUENCE : stock_quants
CREATE SEQUENCE public.stock_quants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_quants
ALTER SEQUENCE public.stock_quants_id_seq OWNED BY public.stock_quants.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_routes
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   sequence:                  integer                    default=10
--   active:                    boolean                    default=true
--   company_id:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_routes
CREATE TABLE public.stock_routes (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint
);

-- CREATE SEQUENCE : stock_routes
CREATE SEQUENCE public.stock_routes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_routes
ALTER SEQUENCE public.stock_routes_id_seq OWNED BY public.stock_routes.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_rules
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   action:                    varchar(32)                NOT NULL
--   route_id:                  bigint                     NOT NULL
--   location_src_id:           bigint                     
--   location_dest_id:          bigint                     NOT NULL
--   picking_type_id:           bigint                     NOT NULL
--   procure_method:            varchar(32)                default='make_to_stock'
--   warehouse_id:              bigint                     
--   company_id:                bigint                     NOT NULL
--   sequence:                  integer                    default=10
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_rules
CREATE TABLE public.stock_rules (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    action character varying(32) NOT NULL,
    route_id bigint NOT NULL,
    location_src_id bigint,
    location_dest_id bigint NOT NULL,
    picking_type_id bigint NOT NULL,
    procure_method character varying(32) DEFAULT 'make_to_stock'::character varying NOT NULL,
    warehouse_id bigint,
    company_id bigint NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : stock_rules
CREATE SEQUENCE public.stock_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_rules
ALTER SEQUENCE public.stock_rules_id_seq OWNED BY public.stock_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_storage_categories
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   max_weight:                numeric(10,2)              default=0
--   allow_new_product:         varchar(16)                default='same'
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_storage_categories
CREATE TABLE public.stock_storage_categories (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    max_weight numeric(10,2) DEFAULT 0 NOT NULL,
    allow_new_product character varying(16) DEFAULT 'same'::character varying NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : stock_storage_categories
CREATE SEQUENCE public.stock_storage_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_storage_categories
ALTER SEQUENCE public.stock_storage_categories_id_seq OWNED BY public.stock_storage_categories.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_storage_category_capacities
--   id:                        bigint                     PK NOT NULL
--   storage_category_id:       bigint                     NOT NULL
--   package_type_id:           bigint                     NOT NULL
--   quantity:                  integer                    default=0
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_storage_category_capacities
CREATE TABLE public.stock_storage_category_capacities (
    id bigint NOT NULL,
    storage_category_id bigint NOT NULL,
    package_type_id bigint NOT NULL,
    quantity integer DEFAULT 0 NOT NULL
);

-- CREATE SEQUENCE : stock_storage_category_capacities
CREATE SEQUENCE public.stock_storage_category_capacities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_storage_category_capacities
ALTER SEQUENCE public.stock_storage_category_capacities_id_seq OWNED BY public.stock_storage_category_capacities.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_valuation_adjustment_lines
--   id:                        bigint                     PK NOT NULL
--   landed_cost_id:            bigint                     NOT NULL
--   cost_line_id:              bigint                     
--   move_id:                   bigint                     
--   product_id:                bigint                     
--   quantity:                  numeric(15,4)              default=0.0
--   weight:                    numeric(15,4)              default=0.0
--   volume:                    numeric(15,4)              default=0.0
--   former_cost:               numeric(15,4)              default=0.0
--   additional_cost:           numeric(15,4)              default=0.0
--   final_cost:                numeric(15,4)              default=0.0
--   move_remaining_qty:        numeric(15,4)              default=0.0
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_valuation_adjustment_lines
CREATE TABLE public.stock_valuation_adjustment_lines (
    id bigint NOT NULL,
    landed_cost_id bigint NOT NULL,
    cost_line_id bigint,
    move_id bigint,
    product_id bigint,
    quantity numeric(15,4) DEFAULT 0.0 NOT NULL,
    weight numeric(15,4) DEFAULT 0.0 NOT NULL,
    volume numeric(15,4) DEFAULT 0.0 NOT NULL,
    former_cost numeric(15,4) DEFAULT 0.0 NOT NULL,
    additional_cost numeric(15,4) DEFAULT 0.0 NOT NULL,
    final_cost numeric(15,4) DEFAULT 0.0 NOT NULL,
    move_remaining_qty numeric(15,4) DEFAULT 0.0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : stock_valuation_adjustment_lines
CREATE SEQUENCE public.stock_valuation_adjustment_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_valuation_adjustment_lines
ALTER SEQUENCE public.stock_valuation_adjustment_lines_id_seq OWNED BY public.stock_valuation_adjustment_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.stock_warehouses
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   code:                      varchar(16)                NOT NULL
--   company_id:                bigint                     
--   partner_id:                bigint                     
--   view_location_id:          bigint                     
--   lot_stock_id:              bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.stock_warehouses
CREATE TABLE public.stock_warehouses (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    code character varying(16) NOT NULL,
    company_id bigint,
    partner_id bigint,
    view_location_id bigint,
    lot_stock_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : stock_warehouses
CREATE SEQUENCE public.stock_warehouses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : stock_warehouses
ALTER SEQUENCE public.stock_warehouses_id_seq OWNED BY public.stock_warehouses.id;

-- ==============================================================================
-- SECTION: products   Product & UoM (المنتجات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 11
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.product_attribute_values
--   id:                        bigint                     PK NOT NULL
--   attribute_id:              bigint                     NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   extra_price:               numeric(15,4)              default=0.0
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_attribute_values
CREATE TABLE public.product_attribute_values (
    id bigint NOT NULL,
    attribute_id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    extra_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : product_attribute_values
CREATE SEQUENCE public.product_attribute_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_attribute_values
ALTER SEQUENCE public.product_attribute_values_id_seq OWNED BY public.product_attribute_values.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_attributes
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_attributes
CREATE TABLE public.product_attributes (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : product_attributes
CREATE SEQUENCE public.product_attributes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_attributes
ALTER SEQUENCE public.product_attributes_id_seq OWNED BY public.product_attributes.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_categories
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   parent_id:                 bigint                     
--   complete_name:             varchar(500)               NOT NULL
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   property_cost_method:      varchar(10)                
--   property_valuation:        varchar(10)                
--   property_lot_valuated:     boolean                    
--   property_stock_valuation_account_id: bigint                     
--   property_price_difference_account_id: bigint                     
--   property_stock_journal_id: bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_categories
CREATE TABLE public.product_categories (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    parent_id bigint,
    complete_name character varying(500) NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    property_cost_method character varying(10),
    property_valuation character varying(10),
    property_lot_valuated boolean,
    property_stock_valuation_account_id bigint,
    property_price_difference_account_id bigint,
    property_stock_journal_id bigint
);

-- CREATE SEQUENCE : product_categories
CREATE SEQUENCE public.product_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_categories
ALTER SEQUENCE public.product_categories_id_seq OWNED BY public.product_categories.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_packagings
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   product_id:                bigint                     NOT NULL
--   barcode:                   varchar(128)               
--   qty:                       numeric(15,4)              default=1
--   package_type_id:           bigint                     
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_packagings
CREATE TABLE public.product_packagings (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    product_id bigint NOT NULL,
    barcode character varying(128),
    qty numeric(15,4) DEFAULT 1 NOT NULL,
    package_type_id bigint,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : product_packagings
CREATE SEQUENCE public.product_packagings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_packagings
ALTER SEQUENCE public.product_packagings_id_seq OWNED BY public.product_packagings.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_pricelist_items
--   id:                        bigint                     PK NOT NULL
--   pricelist_id:              bigint                     NOT NULL
--   applied_on:                varchar(20)                default='all'
--   category_id:               bigint                     
--   template_id:               bigint                     
--   variant_id:                bigint                     
--   min_quantity:              numeric(15,4)              default=1.0
--   compute_price:             varchar(20)                default='fixed'
--   fixed_price:               numeric(15,4)              default=0.0
--   percent_price:             numeric(15,4)              default=0.0
--   date_start:                timestamptz                
--   date_end:                  timestamptz                
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_pricelist_items
CREATE TABLE public.product_pricelist_items (
    id bigint NOT NULL,
    pricelist_id bigint NOT NULL,
    applied_on character varying(20) DEFAULT 'all'::character varying NOT NULL,
    category_id bigint,
    template_id bigint,
    variant_id bigint,
    min_quantity numeric(15,4) DEFAULT 1.0 NOT NULL,
    compute_price character varying(20) DEFAULT 'fixed'::character varying NOT NULL,
    fixed_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    percent_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    date_start timestamp with time zone,
    date_end timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : product_pricelist_items
CREATE SEQUENCE public.product_pricelist_items_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_pricelist_items
ALTER SEQUENCE public.product_pricelist_items_id_seq OWNED BY public.product_pricelist_items.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_pricelists
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   currency:                  varchar(10)                default='USD'
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_pricelists
CREATE TABLE public.product_pricelists (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : product_pricelists
CREATE SEQUENCE public.product_pricelists_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_pricelists
ALTER SEQUENCE public.product_pricelists_id_seq OWNED BY public.product_pricelists.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_templates
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   type:                      varchar(20)                default='consu'
--   category_id:               bigint                     
--   internal_ref:              varchar(100)               
--   barcode:                   varchar(100)               
--   sale_price:                numeric(15,4)              default=0.0
--   cost_price:                numeric(15,4)              default=0.0
--   uom_id:                    bigint                     
--   sale_ok:                   boolean                    default=true
--   purchase_ok:               boolean                    default=true
--   weight:                    numeric(15,4)              default=0.0
--   volume:                    numeric(15,4)              default=0.0
--   description:               text                       
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   cost_method:               varchar(10)                default='standard'
--   valuation:                 varchar(10)                default='real_time'
--   lot_valuated:              boolean                    default=false
--   avg_cost:                  numeric(15,4)              default=0.0
--   total_value:               numeric(15,4)              default=0.0
--   stock_valuation_account_id: bigint                     
--   price_difference_account_id: bigint                     
--   stock_journal_id:          bigint                     
--   landed_cost_ok:            boolean                    default=false
--   split_method_landed_cost:  varchar(24)                default='equal'
--   tracking:                  varchar(16)                default='none'
--   CONSTRAINT:                product_templates_tracking_check CHECK (((tracking)::text = ANY ((ARRAY['none'::varchar, 'lot'::varchar, 'serial'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_templates
CREATE TABLE public.product_templates (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    type character varying(20) DEFAULT 'consu'::character varying NOT NULL,
    category_id bigint,
    internal_ref character varying(100),
    barcode character varying(100),
    sale_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    cost_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    uom_id bigint,
    sale_ok boolean DEFAULT true NOT NULL,
    purchase_ok boolean DEFAULT true NOT NULL,
    weight numeric(15,4) DEFAULT 0.0 NOT NULL,
    volume numeric(15,4) DEFAULT 0.0 NOT NULL,
    description text,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    cost_method character varying(10) DEFAULT 'standard'::character varying NOT NULL,
    valuation character varying(10) DEFAULT 'real_time'::character varying NOT NULL,
    lot_valuated boolean DEFAULT false NOT NULL,
    avg_cost numeric(15,4) DEFAULT 0.0 NOT NULL,
    total_value numeric(15,4) DEFAULT 0.0 NOT NULL,
    stock_valuation_account_id bigint,
    price_difference_account_id bigint,
    stock_journal_id bigint,
    landed_cost_ok boolean DEFAULT false NOT NULL,
    split_method_landed_cost character varying(24) DEFAULT 'equal'::character varying NOT NULL,
    tracking character varying(16) DEFAULT 'none'::character varying NOT NULL,
    CONSTRAINT product_templates_tracking_check CHECK (((tracking)::text = ANY ((ARRAY['none'::character varying, 'lot'::character varying, 'serial'::character varying])::text[])))
);

-- CREATE SEQUENCE : product_templates
CREATE SEQUENCE public.product_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_templates
ALTER SEQUENCE public.product_templates_id_seq OWNED BY public.product_templates.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_values
--   id:                        bigint                     PK NOT NULL
--   product_id:                bigint                     
--   lot_id:                    bigint                     
--   move_id:                   bigint                     
--   value:                     numeric(15,4)              default=0.0
--   company_id:                bigint                     
--   date:                      timestamptz                default=now()
--   user_id:                   bigint                     
--   description:               text                       
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_values
CREATE TABLE public.product_values (
    id bigint NOT NULL,
    product_id bigint,
    lot_id bigint,
    move_id bigint,
    value numeric(15,4) DEFAULT 0.0 NOT NULL,
    company_id bigint,
    date timestamp with time zone DEFAULT now() NOT NULL,
    user_id bigint,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : product_values
CREATE SEQUENCE public.product_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_values
ALTER SEQUENCE public.product_values_id_seq OWNED BY public.product_values.id;

-- ----------------------------------------------------------------------
-- TABLE: public.product_variant_attributes
--   variant_id:                bigint                     PK NOT NULL
--   attribute_value_id:        bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_variant_attributes
CREATE TABLE public.product_variant_attributes (
    variant_id bigint NOT NULL,
    attribute_value_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.product_variants
--   id:                        bigint                     PK NOT NULL
--   template_id:               bigint                     NOT NULL
--   sku:                       varchar(100)               
--   barcode:                   varchar(100)               
--   extra_price:               numeric(15,4)              default=0.0
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.product_variants
CREATE TABLE public.product_variants (
    id bigint NOT NULL,
    template_id bigint NOT NULL,
    sku character varying(100),
    barcode character varying(100),
    extra_price numeric(15,4) DEFAULT 0.0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : product_variants
CREATE SEQUENCE public.product_variants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : product_variants
ALTER SEQUENCE public.product_variants_id_seq OWNED BY public.product_variants.id;

-- ----------------------------------------------------------------------
-- TABLE: public.uom_uoms
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   category:                  varchar(50)                NOT NULL
--   ratio:                     numeric(15,6)              default=1.0
--   rounding:                  numeric(15,6)              default=0.001
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.uom_uoms
CREATE TABLE public.uom_uoms (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    category character varying(50) NOT NULL,
    ratio numeric(15,6) DEFAULT 1.0 NOT NULL,
    rounding numeric(15,6) DEFAULT 0.001 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : uom_uoms
CREATE SEQUENCE public.uom_uoms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : uom_uoms
ALTER SEQUENCE public.uom_uoms_id_seq OWNED BY public.uom_uoms.id;

-- ==============================================================================
-- SECTION: sales      Sales, Loyalty, eCommerce (المبيعات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 18
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.ecommerce_cart_lines
--   id:                        bigint                     PK NOT NULL
--   cart_id:                   bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   quantity:                  numeric(15,4)              default=1
--   price_unit:                numeric(15,4)              default=0
--   discount:                  numeric(5,2)               default=0
--   price_total:               numeric(15,4)              default=0
--   tax_ids:                   bigint[]                   default=ARRAY[]::bigint[]
--   notes:                     text                       
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ecommerce_cart_lines
CREATE TABLE public.ecommerce_cart_lines (
    id bigint NOT NULL,
    cart_id bigint NOT NULL,
    product_id bigint NOT NULL,
    quantity numeric(15,4) DEFAULT 1 NOT NULL,
    price_unit numeric(15,4) DEFAULT 0 NOT NULL,
    discount numeric(5,2) DEFAULT 0 NOT NULL,
    price_total numeric(15,4) DEFAULT 0 NOT NULL,
    tax_ids bigint[] DEFAULT ARRAY[]::bigint[],
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : ecommerce_cart_lines
CREATE SEQUENCE public.ecommerce_cart_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ecommerce_cart_lines
ALTER SEQUENCE public.ecommerce_cart_lines_id_seq OWNED BY public.ecommerce_cart_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.ecommerce_carts
--   id:                        bigint                     PK NOT NULL
--   website_id:                bigint                     NOT NULL
--   session_uuid:              varchar(64)                NOT NULL
--   partner_id:                bigint                     
--   pricelist_id:              bigint                     NOT NULL
--   currency:                  varchar(3)                 default='SAR'
--   state:                     varchar(32)                default='active'
--   shipping_address_id:       bigint                     
--   invoice_address_id:        bigint                     
--   delivery_method_id:        bigint                     
--   shipping_amount:           numeric(15,4)              default=0
--   coupon_code:               varchar(64)                
--   discount_amount:           numeric(15,4)              default=0
--   amount_untaxed:            numeric(15,4)              default=0
--   amount_tax:                numeric(15,4)              default=0
--   amount_total:              numeric(15,4)              default=0
--   converted_order_id:        bigint                     
--   last_activity_at:          timestamptz                default=now()
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ecommerce_carts
CREATE TABLE public.ecommerce_carts (
    id bigint NOT NULL,
    website_id bigint NOT NULL,
    session_uuid character varying(64) NOT NULL,
    partner_id bigint,
    pricelist_id bigint NOT NULL,
    currency character varying(3) DEFAULT 'SAR'::character varying NOT NULL,
    state character varying(32) DEFAULT 'active'::character varying NOT NULL,
    shipping_address_id bigint,
    invoice_address_id bigint,
    delivery_method_id bigint,
    shipping_amount numeric(15,4) DEFAULT 0 NOT NULL,
    coupon_code character varying(64),
    discount_amount numeric(15,4) DEFAULT 0 NOT NULL,
    amount_untaxed numeric(15,4) DEFAULT 0 NOT NULL,
    amount_tax numeric(15,4) DEFAULT 0 NOT NULL,
    amount_total numeric(15,4) DEFAULT 0 NOT NULL,
    converted_order_id bigint,
    last_activity_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : ecommerce_carts
CREATE SEQUENCE public.ecommerce_carts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ecommerce_carts
ALTER SEQUENCE public.ecommerce_carts_id_seq OWNED BY public.ecommerce_carts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_card_history
--   id:                        bigint                     PK NOT NULL
--   card_id:                   bigint                     NOT NULL
--   company_id:                bigint                     
--   description:               text                       NOT NULL
--   issued:                    numeric(15,4)              default=0.0000
--   used:                      numeric(15,4)              default=0.0000
--   order_model:               varchar(32)                
--   order_id:                  bigint                     
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_card_history
CREATE TABLE public.loyalty_card_history (
    id bigint NOT NULL,
    card_id bigint NOT NULL,
    company_id bigint,
    description text NOT NULL,
    issued numeric(15,4) DEFAULT 0.0000 NOT NULL,
    used numeric(15,4) DEFAULT 0.0000 NOT NULL,
    order_model character varying(32),
    order_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : loyalty_card_history
CREATE SEQUENCE public.loyalty_card_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_card_history
ALTER SEQUENCE public.loyalty_card_history_id_seq OWNED BY public.loyalty_card_history.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_cards
--   id:                        bigint                     PK NOT NULL
--   program_id:                bigint                     NOT NULL
--   company_id:                bigint                     
--   partner_id:                bigint                     
--   points:                    numeric(15,4)              default=0.0000
--   code:                      varchar(64)                NOT NULL
--   expiration_date:           timestamptz                
--   use_count:                 integer                    default=0
--   order_id:                  bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_cards
CREATE TABLE public.loyalty_cards (
    id bigint NOT NULL,
    program_id bigint NOT NULL,
    company_id bigint,
    partner_id bigint,
    points numeric(15,4) DEFAULT 0.0000 NOT NULL,
    code character varying(64) NOT NULL,
    expiration_date timestamp with time zone,
    use_count integer DEFAULT 0 NOT NULL,
    order_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : loyalty_cards
CREATE SEQUENCE public.loyalty_cards_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_cards
ALTER SEQUENCE public.loyalty_cards_id_seq OWNED BY public.loyalty_cards.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_mails
--   id:                        bigint                     PK NOT NULL
--   active:                    boolean                    default=true
--   program_id:                bigint                     NOT NULL
--   trigger:                   varchar(16)                default='create'
--   points:                    numeric(15,4)              default=0.0000
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_mails
CREATE TABLE public.loyalty_mails (
    id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    program_id bigint NOT NULL,
    trigger character varying(16) DEFAULT 'create'::character varying NOT NULL,
    points numeric(15,4) DEFAULT 0.0000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : loyalty_mails
CREATE SEQUENCE public.loyalty_mails_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_mails
ALTER SEQUENCE public.loyalty_mails_id_seq OWNED BY public.loyalty_mails.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_program_pricelists
--   program_id:                bigint                     PK NOT NULL
--   pricelist_id:              bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_program_pricelists
CREATE TABLE public.loyalty_program_pricelists (
    program_id bigint NOT NULL,
    pricelist_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_programs
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   active:                    boolean                    default=true
--   sequence:                  integer                    default=10
--   company_id:                bigint                     
--   currency:                  varchar(10)                default='USD'
--   program_type:              varchar(20)                default='promotion'
--   date_from:                 timestamptz                
--   date_to:                   timestamptz                
--   limit_usage:               boolean                    default=false
--   max_usage:                 integer                    default=0
--   applies_on:                varchar(10)                default='current'
--   trigger:                   varchar(10)                default='auto'
--   portal_visible:            boolean                    default=false
--   portal_point_name:         varchar(64)                default='Points'
--   is_nominative:             boolean                    default=false
--   is_payment_program:        boolean                    default=false
--   sale_ok:                   boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_programs
CREATE TABLE public.loyalty_programs (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    company_id bigint,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    program_type character varying(20) DEFAULT 'promotion'::character varying NOT NULL,
    date_from timestamp with time zone,
    date_to timestamp with time zone,
    limit_usage boolean DEFAULT false NOT NULL,
    max_usage integer DEFAULT 0 NOT NULL,
    applies_on character varying(10) DEFAULT 'current'::character varying NOT NULL,
    trigger character varying(10) DEFAULT 'auto'::character varying NOT NULL,
    portal_visible boolean DEFAULT false NOT NULL,
    portal_point_name character varying(64) DEFAULT 'Points'::character varying NOT NULL,
    is_nominative boolean DEFAULT false NOT NULL,
    is_payment_program boolean DEFAULT false NOT NULL,
    sale_ok boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : loyalty_programs
CREATE SEQUENCE public.loyalty_programs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_programs
ALTER SEQUENCE public.loyalty_programs_id_seq OWNED BY public.loyalty_programs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_rewards
--   id:                        bigint                     PK NOT NULL
--   active:                    boolean                    default=true
--   program_id:                bigint                     NOT NULL
--   description:               jsonb                      
--   reward_type:               varchar(10)                default='discount'
--   discount:                  numeric(15,4)              default=0.0000
--   discount_mode:             varchar(10)                default='percent'
--   discount_applicability:    varchar(10)                default='order'
--   discount_product_ids:      bigint[]                   default='{}'::bigint[]
--   discount_product_category_id: bigint                     
--   discount_product_tag_id:   bigint                     
--   discount_max_amount:       numeric(15,4)              default=0.0000
--   discount_line_product_id:  bigint                     
--   reward_product_id:         bigint                     
--   reward_product_qty:        integer                    default=1
--   reward_product_uom_id:     bigint                     
--   required_points:           numeric(15,4)              default=1.0000
--   clear_wallet:              boolean                    default=false
--   product_domain:            text                       
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_rewards
CREATE TABLE public.loyalty_rewards (
    id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    program_id bigint NOT NULL,
    description jsonb,
    reward_type character varying(10) DEFAULT 'discount'::character varying NOT NULL,
    discount numeric(15,4) DEFAULT 0.0000 NOT NULL,
    discount_mode character varying(10) DEFAULT 'percent'::character varying NOT NULL,
    discount_applicability character varying(10) DEFAULT 'order'::character varying NOT NULL,
    discount_product_ids bigint[] DEFAULT '{}'::bigint[],
    discount_product_category_id bigint,
    discount_product_tag_id bigint,
    discount_max_amount numeric(15,4) DEFAULT 0.0000 NOT NULL,
    discount_line_product_id bigint,
    reward_product_id bigint,
    reward_product_qty integer DEFAULT 1 NOT NULL,
    reward_product_uom_id bigint,
    required_points numeric(15,4) DEFAULT 1.0000 NOT NULL,
    clear_wallet boolean DEFAULT false NOT NULL,
    product_domain text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : loyalty_rewards
CREATE SEQUENCE public.loyalty_rewards_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_rewards
ALTER SEQUENCE public.loyalty_rewards_id_seq OWNED BY public.loyalty_rewards.id;

-- ----------------------------------------------------------------------
-- TABLE: public.loyalty_rules
--   id:                        bigint                     PK NOT NULL
--   active:                    boolean                    default=true
--   program_id:                bigint                     NOT NULL
--   company_id:                bigint                     
--   product_ids:               bigint[]                   default='{}'::bigint[]
--   product_category_id:       bigint                     
--   product_tag_id:            bigint                     
--   product_domain:            text                       
--   reward_point_amount:       numeric(15,4)              default=1.0000
--   reward_point_split:        boolean                    default=false
--   reward_point_mode:         varchar(8)                 default='order'
--   minimum_qty:               integer                    default=1
--   minimum_amount:            numeric(15,4)              default=0.0000
--   minimum_amount_tax_mode:   varchar(4)                 default='incl'
--   mode:                      varchar(10)                default='auto'
--   code:                      varchar(64)                
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.loyalty_rules
CREATE TABLE public.loyalty_rules (
    id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    program_id bigint NOT NULL,
    company_id bigint,
    product_ids bigint[] DEFAULT '{}'::bigint[],
    product_category_id bigint,
    product_tag_id bigint,
    product_domain text,
    reward_point_amount numeric(15,4) DEFAULT 1.0000 NOT NULL,
    reward_point_split boolean DEFAULT false NOT NULL,
    reward_point_mode character varying(8) DEFAULT 'order'::character varying NOT NULL,
    minimum_qty integer DEFAULT 1 NOT NULL,
    minimum_amount numeric(15,4) DEFAULT 0.0000 NOT NULL,
    minimum_amount_tax_mode character varying(4) DEFAULT 'incl'::character varying NOT NULL,
    mode character varying(10) DEFAULT 'auto'::character varying NOT NULL,
    code character varying(64),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : loyalty_rules
CREATE SEQUENCE public.loyalty_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : loyalty_rules
ALTER SEQUENCE public.loyalty_rules_id_seq OWNED BY public.loyalty_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.sale_order_coupon_points
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   coupon_id:                 bigint                     NOT NULL
--   points:                    numeric(15,4)              default=0.0000
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.sale_order_coupon_points
CREATE TABLE public.sale_order_coupon_points (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    coupon_id bigint NOT NULL,
    points numeric(15,4) DEFAULT 0.0000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : sale_order_coupon_points
CREATE SEQUENCE public.sale_order_coupon_points_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : sale_order_coupon_points
ALTER SEQUENCE public.sale_order_coupon_points_id_seq OWNED BY public.sale_order_coupon_points.id;

-- ----------------------------------------------------------------------
-- TABLE: public.sale_order_invoices
--   order_id:                  bigint                     PK NOT NULL
--   move_id:                   bigint                     PK NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.sale_order_invoices
CREATE TABLE public.sale_order_invoices (
    order_id bigint NOT NULL,
    move_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.sale_order_lines
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   sequence:                  integer                    default=10
--   product_id:                bigint                     NOT NULL
--   name:                      jsonb                      NOT NULL
--   product_uom_qty:           numeric(15,4)              default=1.0000
--   product_uom:               bigint                     
--   price_unit:                numeric(15,4)              default=0.0000
--   discount:                  numeric(5,2)               default=0.00
--   tax_ids:                   bigint[]                   default='{}'::bigint[]
--   price_subtotal:            numeric(15,4)              default=0.0000
--   price_tax:                 numeric(15,4)              default=0.0000
--   price_total:               numeric(15,4)              default=0.0000
--   qty_delivered:             numeric(15,4)              default=0.0000
--   qty_invoiced:              numeric(15,4)              default=0.0000
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   route_id:                  bigint                     
--   reward_id:                 bigint                     
--   coupon_id:                 bigint                     
--   reward_identifier_code:    varchar(64)                
--   points_cost:               numeric(15,4)              default=0.0000
--   is_reward_line:            boolean                    default=false
--   is_delivery:               boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.sale_order_lines
CREATE TABLE public.sale_order_lines (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    product_id bigint NOT NULL,
    name jsonb NOT NULL,
    product_uom_qty numeric(15,4) DEFAULT 1.0000 NOT NULL,
    product_uom bigint,
    price_unit numeric(15,4) DEFAULT 0.0000 NOT NULL,
    discount numeric(5,2) DEFAULT 0.00 NOT NULL,
    tax_ids bigint[] DEFAULT '{}'::bigint[],
    price_subtotal numeric(15,4) DEFAULT 0.0000 NOT NULL,
    price_tax numeric(15,4) DEFAULT 0.0000 NOT NULL,
    price_total numeric(15,4) DEFAULT 0.0000 NOT NULL,
    qty_delivered numeric(15,4) DEFAULT 0.0000 NOT NULL,
    qty_invoiced numeric(15,4) DEFAULT 0.0000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    route_id bigint,
    reward_id bigint,
    coupon_id bigint,
    reward_identifier_code character varying(64),
    points_cost numeric(15,4) DEFAULT 0.0000 NOT NULL,
    is_reward_line boolean DEFAULT false NOT NULL,
    is_delivery boolean DEFAULT false
);

-- CREATE SEQUENCE : sale_order_lines
CREATE SEQUENCE public.sale_order_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : sale_order_lines
ALTER SEQUENCE public.sale_order_lines_id_seq OWNED BY public.sale_order_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.sale_orders
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   partner_id:                bigint                     NOT NULL
--   date_order:                timestamptz                default=now()
--   validity_date:             timestamptz                
--   state:                     varchar(20)                default='draft'
--   invoice_status:            varchar(20)                default='no'
--   pricelist_id:              bigint                     
--   payment_term_id:           bigint                     
--   user_id:                   bigint                     
--   company_id:                bigint                     
--   currency:                  varchar(10)                default='USD'
--   note:                      text                       
--   amount_untaxed:            numeric(15,4)              default=0.0000
--   amount_tax:                numeric(15,4)              default=0.0000
--   amount_total:              numeric(15,4)              default=0.0000
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   delivery_status:           varchar(20)                default='nothing'
--   procurement_group_id:      bigint                     
--   applied_coupon_ids:        bigint[]                   default='{}'::bigint[]
--   code_enabled_rule_ids:     bigint[]                   default='{}'::bigint[]
--   carrier_id:                integer                    
--   shipping_weight:           numeric(19,4)              default=0
--   delivery_message:          text                       
--   recompute_delivery_price:  boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.sale_orders
CREATE TABLE public.sale_orders (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    partner_id bigint NOT NULL,
    date_order timestamp with time zone DEFAULT now() NOT NULL,
    validity_date timestamp with time zone,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    invoice_status character varying(20) DEFAULT 'no'::character varying NOT NULL,
    pricelist_id bigint,
    payment_term_id bigint,
    user_id bigint,
    company_id bigint,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    note text,
    amount_untaxed numeric(15,4) DEFAULT 0.0000 NOT NULL,
    amount_tax numeric(15,4) DEFAULT 0.0000 NOT NULL,
    amount_total numeric(15,4) DEFAULT 0.0000 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    delivery_status character varying(20) DEFAULT 'nothing'::character varying NOT NULL,
    procurement_group_id bigint,
    applied_coupon_ids bigint[] DEFAULT '{}'::bigint[],
    code_enabled_rule_ids bigint[] DEFAULT '{}'::bigint[],
    carrier_id integer,
    shipping_weight numeric(19,4) DEFAULT 0,
    delivery_message text,
    recompute_delivery_price boolean DEFAULT false
);

-- CREATE SEQUENCE : sale_orders
CREATE SEQUENCE public.sale_order_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : sale_orders
CREATE SEQUENCE public.sale_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : sale_orders
ALTER SEQUENCE public.sale_orders_id_seq OWNED BY public.sale_orders.id;

-- ----------------------------------------------------------------------
-- TABLE: public.sale_subscriptions
--   id:                        bigint                     PK NOT NULL
--   code:                      varchar(64)                NOT NULL
--   partner_id:                bigint                     NOT NULL
--   plan_id:                   bigint                     NOT NULL
--   state:                     varchar(32)                default='draft'
--   start_date:                timestamptz                default=now()
--   next_billing_date:         timestamptz                NOT NULL
--   end_date:                  timestamptz                
--   recurring_amount:          numeric(15,4)              default=0
--   payment_token_id:          bigint                     
--   failed_charge_count:       integer                    default=0
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.sale_subscriptions
CREATE TABLE public.sale_subscriptions (
    id bigint NOT NULL,
    code character varying(64) NOT NULL,
    partner_id bigint NOT NULL,
    plan_id bigint NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    start_date timestamp with time zone DEFAULT now() NOT NULL,
    next_billing_date timestamp with time zone NOT NULL,
    end_date timestamp with time zone,
    recurring_amount numeric(15,4) DEFAULT 0 NOT NULL,
    payment_token_id bigint,
    failed_charge_count integer DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : sale_subscriptions
CREATE SEQUENCE public.sale_subscriptions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : sale_subscriptions
ALTER SEQUENCE public.sale_subscriptions_id_seq OWNED BY public.sale_subscriptions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.subscription_plans
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   period:                    varchar(16)                NOT NULL
--   period_interval:           integer                    default=1
--   price:                     numeric(15,4)              default=0
--   currency:                  varchar(3)                 default='SAR'
--   product_id:                bigint                     NOT NULL
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.subscription_plans
CREATE TABLE public.subscription_plans (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    period character varying(16) NOT NULL,
    period_interval integer DEFAULT 1 NOT NULL,
    price numeric(15,4) DEFAULT 0 NOT NULL,
    currency character varying(3) DEFAULT 'SAR'::character varying NOT NULL,
    product_id bigint NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : subscription_plans
CREATE SEQUENCE public.subscription_plans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : subscription_plans
ALTER SEQUENCE public.subscription_plans_id_seq OWNED BY public.subscription_plans.id;

-- ----------------------------------------------------------------------
-- TABLE: public.website_menus
--   id:                        bigint                     PK NOT NULL
--   website_id:                bigint                     NOT NULL
--   parent_id:                 bigint                     
--   name:                      varchar(128)               NOT NULL
--   url:                       varchar(512)               NOT NULL
--   sequence:                  integer                    default=10
--   new_window:                boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.website_menus
CREATE TABLE public.website_menus (
    id bigint NOT NULL,
    website_id bigint NOT NULL,
    parent_id bigint,
    name character varying(128) NOT NULL,
    url character varying(512) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    new_window boolean DEFAULT false NOT NULL
);

-- CREATE SEQUENCE : website_menus
CREATE SEQUENCE public.website_menus_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : website_menus
ALTER SEQUENCE public.website_menus_id_seq OWNED BY public.website_menus.id;

-- ----------------------------------------------------------------------
-- TABLE: public.website_pages
--   id:                        bigint                     PK NOT NULL
--   website_id:                bigint                     NOT NULL
--   title:                     varchar(256)               NOT NULL
--   slug:                      varchar(256)               NOT NULL
--   content_json:              jsonb                      default='{}'
--   meta_title:                varchar(256)               
--   meta_description:          text                       
--   meta_keywords:             varchar(256)               
--   is_published:              boolean                    default=false
--   is_homepage:               boolean                    default=false
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.website_pages
CREATE TABLE public.website_pages (
    id bigint NOT NULL,
    website_id bigint NOT NULL,
    title character varying(256) NOT NULL,
    slug character varying(256) NOT NULL,
    content_json jsonb DEFAULT '{}'::jsonb NOT NULL,
    meta_title character varying(256),
    meta_description text,
    meta_keywords character varying(256),
    is_published boolean DEFAULT false NOT NULL,
    is_homepage boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : website_pages
CREATE SEQUENCE public.website_pages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : website_pages
ALTER SEQUENCE public.website_pages_id_seq OWNED BY public.website_pages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.website_sites
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   domain:                    varchar(256)               NOT NULL
--   company_id:                bigint                     NOT NULL
--   default_language:          varchar(10)                default='ar'
--   supported_langs:           text[]                     default=ARRAY['ar'::text, 'en'::text]
--   pricelist_id:              bigint                     NOT NULL
--   warehouse_id:              bigint                     NOT NULL
--   header_logo_url:           text                       
--   favicon_url:               text                       
--   google_analytics_id:       varchar(64)                
--   theme_config:              jsonb                      default='{}'
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.website_sites
CREATE TABLE public.website_sites (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    domain character varying(256) NOT NULL,
    company_id bigint NOT NULL,
    default_language character varying(10) DEFAULT 'ar'::character varying NOT NULL,
    supported_langs text[] DEFAULT ARRAY['ar'::text, 'en'::text] NOT NULL,
    pricelist_id bigint NOT NULL,
    warehouse_id bigint NOT NULL,
    header_logo_url text,
    favicon_url text,
    google_analytics_id character varying(64),
    theme_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : website_sites
CREATE SEQUENCE public.website_sites_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : website_sites
ALTER SEQUENCE public.website_sites_id_seq OWNED BY public.website_sites.id;

-- ==============================================================================
-- SECTION: purchase   Purchase (المشتريات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 6
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_order_group_members
--   group_id:                  bigint                     PK NOT NULL
--   order_id:                  bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_order_group_members
CREATE TABLE public.purchase_order_group_members (
    group_id bigint NOT NULL,
    order_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_order_groups
--   id:                        bigint                     PK NOT NULL
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_order_groups
CREATE TABLE public.purchase_order_groups (
    id bigint NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : purchase_order_groups
CREATE SEQUENCE public.purchase_order_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_order_groups
ALTER SEQUENCE public.purchase_order_groups_id_seq OWNED BY public.purchase_order_groups.id;

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_order_invoices
--   order_id:                  bigint                     PK NOT NULL
--   move_id:                   bigint                     PK NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_order_invoices
CREATE TABLE public.purchase_order_invoices (
    order_id bigint NOT NULL,
    move_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_order_lines
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   sequence:                  integer                    default=10
--   product_id:                bigint                     NOT NULL
--   name:                      text                       NOT NULL
--   product_qty:               numeric(15,4)              default=1.0000
--   product_uom:               bigint                     
--   price_unit:                numeric(15,4)              default=0.0000
--   discount:                  numeric(5,2)               default=0.00
--   tax_ids:                   bigint[]                   default='{}'::bigint[]
--   price_subtotal:            numeric(15,4)              default=0.0000
--   price_tax:                 numeric(15,4)              default=0.0000
--   price_total:               numeric(15,4)              default=0.0000
--   qty_received:              numeric(15,4)              default=0.0000
--   qty_invoiced:              numeric(15,4)              default=0.0000
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_order_lines
CREATE TABLE public.purchase_order_lines (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    product_id bigint NOT NULL,
    name text NOT NULL,
    product_qty numeric(15,4) DEFAULT 1.0000 NOT NULL,
    product_uom bigint,
    price_unit numeric(15,4) DEFAULT 0.0000 NOT NULL,
    discount numeric(5,2) DEFAULT 0.00 NOT NULL,
    tax_ids bigint[] DEFAULT '{}'::bigint[],
    price_subtotal numeric(15,4) DEFAULT 0.0000 NOT NULL,
    price_tax numeric(15,4) DEFAULT 0.0000 NOT NULL,
    price_total numeric(15,4) DEFAULT 0.0000 NOT NULL,
    qty_received numeric(15,4) DEFAULT 0.0000 NOT NULL,
    qty_invoiced numeric(15,4) DEFAULT 0.0000 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : purchase_order_lines
CREATE SEQUENCE public.purchase_order_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_order_lines
ALTER SEQUENCE public.purchase_order_lines_id_seq OWNED BY public.purchase_order_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_orders
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   partner_id:                bigint                     NOT NULL
--   date_order:                timestamptz                default=now()
--   date_planned:              timestamptz                
--   state:                     varchar(20)                default='draft'
--   invoice_status:            varchar(20)                default='no'
--   payment_term_id:           bigint                     
--   user_id:                   bigint                     
--   company_id:                bigint                     
--   currency:                  varchar(10)                default='USD'
--   note:                      text                       
--   amount_untaxed:            numeric(15,4)              default=0.0000
--   amount_tax:                numeric(15,4)              default=0.0000
--   amount_total:              numeric(15,4)              default=0.0000
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   receipt_status:            varchar(20)                default='nothing'
--   procurement_group_id:      bigint                     
--   requisition_id:            bigint                     
--   requisition_type:          varchar(32)                
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_orders
CREATE TABLE public.purchase_orders (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    partner_id bigint NOT NULL,
    date_order timestamp with time zone DEFAULT now() NOT NULL,
    date_planned timestamp with time zone,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    invoice_status character varying(20) DEFAULT 'no'::character varying NOT NULL,
    payment_term_id bigint,
    user_id bigint,
    company_id bigint,
    currency character varying(10) DEFAULT 'USD'::character varying NOT NULL,
    note text,
    amount_untaxed numeric(15,4) DEFAULT 0.0000 NOT NULL,
    amount_tax numeric(15,4) DEFAULT 0.0000 NOT NULL,
    amount_total numeric(15,4) DEFAULT 0.0000 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    receipt_status character varying(20) DEFAULT 'nothing'::character varying NOT NULL,
    procurement_group_id bigint,
    requisition_id bigint,
    requisition_type character varying(32)
);

-- CREATE SEQUENCE : purchase_orders
CREATE SEQUENCE public.purchase_order_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- CREATE SEQUENCE : purchase_orders
CREATE SEQUENCE public.purchase_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_orders
ALTER SEQUENCE public.purchase_orders_id_seq OWNED BY public.purchase_orders.id;

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_supplier_infos
--   id:                        bigint                     PK NOT NULL
--   requisition_id:            bigint                     NOT NULL
--   requisition_line_id:       bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   vendor_id:                 bigint                     NOT NULL
--   product_uom:               bigint                     
--   price:                     numeric(15,4)              NOT NULL
--   currency_id:               bigint                     
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                purchase_supplier_infos_price_check CHECK ((price > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_supplier_infos
CREATE TABLE public.purchase_supplier_infos (
    id bigint NOT NULL,
    requisition_id bigint NOT NULL,
    requisition_line_id bigint NOT NULL,
    product_id bigint NOT NULL,
    vendor_id bigint NOT NULL,
    product_uom bigint,
    price numeric(15,4) NOT NULL,
    currency_id bigint,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT purchase_supplier_infos_price_check CHECK ((price > (0)::numeric))
);

-- CREATE SEQUENCE : purchase_supplier_infos
CREATE SEQUENCE public.purchase_supplier_infos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_supplier_infos
ALTER SEQUENCE public.purchase_supplier_infos_id_seq OWNED BY public.purchase_supplier_infos.id;

-- ==============================================================================
-- SECTION: prm        PRM - Purchase Requisition Management (الطلبات الشرائية)
-- ------------------------------------------------------------------------------
-- Tables in this section: 2
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_requisition_lines
--   id:                        bigint                     PK NOT NULL
--   requisition_id:            bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   product_qty:               numeric(15,4)              default=1.0000
--   product_uom:               bigint                     
--   price_unit:                numeric(15,4)              default=0.0000
--   schedule_date:             timestamptz                
--   supplier_id:               bigint                     
--   description:               text                       
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   qty_ordered:               numeric(15,4)              default=0.0000
--   product_description_variants: varchar(255)               
--   CONSTRAINT:                purchase_requisition_lines_ordered_quantity_check CHECK ((qty_ordered >= (0)::numeric)) 
--   CONSTRAINT:                purchase_requisition_lines_price_check CHECK ((price_unit >= (0)::numeric)) 
--   CONSTRAINT:                purchase_requisition_lines_quantity_check CHECK ((product_qty > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_requisition_lines
CREATE TABLE public.purchase_requisition_lines (
    id bigint NOT NULL,
    requisition_id bigint NOT NULL,
    product_id bigint NOT NULL,
    product_qty numeric(15,4) DEFAULT 1.0000 NOT NULL,
    product_uom bigint,
    price_unit numeric(15,4) DEFAULT 0.0000 NOT NULL,
    schedule_date timestamp with time zone,
    supplier_id bigint,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    qty_ordered numeric(15,4) DEFAULT 0.0000 NOT NULL,
    product_description_variants character varying(255),
    CONSTRAINT purchase_requisition_lines_ordered_quantity_check CHECK ((qty_ordered >= (0)::numeric)),
    CONSTRAINT purchase_requisition_lines_price_check CHECK ((price_unit >= (0)::numeric)),
    CONSTRAINT purchase_requisition_lines_quantity_check CHECK ((product_qty > (0)::numeric))
);

-- CREATE SEQUENCE : purchase_requisition_lines
CREATE SEQUENCE public.purchase_requisition_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_requisition_lines
ALTER SEQUENCE public.purchase_requisition_lines_id_seq OWNED BY public.purchase_requisition_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.purchase_requisitions
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   requisition_type:          varchar(32)                default='blanket_order'
--   vendor_id:                 bigint                     
--   user_id:                   bigint                     
--   date_start:                timestamptz                
--   date_end:                  timestamptz                
--   state:                     varchar(32)                default='draft'
--   currency_id:               bigint                     
--   company_id:                bigint                     
--   description:               text                       
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   active:                    boolean                    default=true
--   reference:                 varchar(255)               
--   order_count:               integer                    default=0
--   CONSTRAINT:                purchase_requisitions_dates_check CHECK (((date_end IS NULL) OR (date_start IS NULL) OR (date_end >= date_start))) 
--   CONSTRAINT:                purchase_requisitions_order_count_check CHECK ((order_count >= 0)) 
--   CONSTRAINT:                purchase_requisitions_state_check CHECK (((state)::text = ANY ((ARRAY['draft'::varchar, 'confirmed'::varchar, 'done'::varchar, 'cancel'::varchar])::text[]))) 
--   CONSTRAINT:                purchase_requisitions_type_check CHECK (((requisition_type)::text = ANY ((ARRAY['blanket_order'::varchar, 'purchase_template'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.purchase_requisitions
CREATE TABLE public.purchase_requisitions (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    requisition_type character varying(32) DEFAULT 'blanket_order'::character varying NOT NULL,
    vendor_id bigint,
    user_id bigint,
    date_start timestamp with time zone,
    date_end timestamp with time zone,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    currency_id bigint,
    company_id bigint,
    description text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    active boolean DEFAULT true NOT NULL,
    reference character varying(255),
    order_count integer DEFAULT 0 NOT NULL,
    CONSTRAINT purchase_requisitions_dates_check CHECK (((date_end IS NULL) OR (date_start IS NULL) OR (date_end >= date_start))),
    CONSTRAINT purchase_requisitions_order_count_check CHECK ((order_count >= 0)),
    CONSTRAINT purchase_requisitions_state_check CHECK (((state)::text = ANY ((ARRAY['draft'::character varying, 'confirmed'::character varying, 'done'::character varying, 'cancel'::character varying])::text[]))),
    CONSTRAINT purchase_requisitions_type_check CHECK (((requisition_type)::text = ANY ((ARRAY['blanket_order'::character varying, 'purchase_template'::character varying])::text[])))
);

-- CREATE SEQUENCE : purchase_requisitions
CREATE SEQUENCE public.purchase_requisitions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : purchase_requisitions
ALTER SEQUENCE public.purchase_requisitions_id_seq OWNED BY public.purchase_requisitions.id;

-- ==============================================================================
-- SECTION: crm        CRM (إدارة علاقات العملاء)
-- ------------------------------------------------------------------------------
-- Tables in this section: 5
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.crm_lead_tags
--   lead_id:                   bigint                     PK NOT NULL
--   tag_id:                    bigint                     PK NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.crm_lead_tags
CREATE TABLE public.crm_lead_tags (
    lead_id bigint NOT NULL,
    tag_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.crm_leads
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   type:                      varchar(20)                default='lead'
--   partner_id:                bigint                     
--   partner_name:              varchar(255)               
--   contact_name:              varchar(255)               
--   email_from:                public.citext              
--   phone:                     varchar(50)                
--   stage_id:                  bigint                     NOT NULL
--   salesperson_id:            bigint                     
--   expected_revenue:          numeric(15,4)              default=0.0000
--   prorated_revenue:          numeric(15,4)              default=0.0000
--   probability:               numeric(5,2)               default=0.00
--   source:                    varchar(64)                
--   priority:                  varchar(20)                default='1'
--   lost_reason_id:            bigint                     
--   lost_feedback:             text                       
--   date_deadline:             timestamptz                
--   date_closed:               timestamptz                
--   date_conversion:           timestamptz                
--   notes:                     text                       
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.crm_leads
CREATE TABLE public.crm_leads (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(20) DEFAULT 'lead'::character varying NOT NULL,
    partner_id bigint,
    partner_name character varying(255),
    contact_name character varying(255),
    email_from public.citext,
    phone character varying(50),
    stage_id bigint NOT NULL,
    salesperson_id bigint,
    expected_revenue numeric(15,4) DEFAULT 0.0000 NOT NULL,
    prorated_revenue numeric(15,4) DEFAULT 0.0000 NOT NULL,
    probability numeric(5,2) DEFAULT 0.00 NOT NULL,
    source character varying(64),
    priority character varying(20) DEFAULT '1'::character varying NOT NULL,
    lost_reason_id bigint,
    lost_feedback text,
    date_deadline timestamp with time zone,
    date_closed timestamp with time zone,
    date_conversion timestamp with time zone,
    notes text,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : crm_leads
CREATE SEQUENCE public.crm_leads_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : crm_leads
ALTER SEQUENCE public.crm_leads_id_seq OWNED BY public.crm_leads.id;

-- ----------------------------------------------------------------------
-- TABLE: public.crm_lost_reasons
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.crm_lost_reasons
CREATE TABLE public.crm_lost_reasons (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : crm_lost_reasons
CREATE SEQUENCE public.crm_lost_reasons_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : crm_lost_reasons
ALTER SEQUENCE public.crm_lost_reasons_id_seq OWNED BY public.crm_lost_reasons.id;

-- ----------------------------------------------------------------------
-- TABLE: public.crm_stages
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   is_won:                    boolean                    default=false
--   is_closed:                 boolean                    default=false
--   fold:                      boolean                    default=false
--   requirements:              text                       
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.crm_stages
CREATE TABLE public.crm_stages (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    is_won boolean DEFAULT false NOT NULL,
    is_closed boolean DEFAULT false NOT NULL,
    fold boolean DEFAULT false NOT NULL,
    requirements text,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : crm_stages
CREATE SEQUENCE public.crm_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : crm_stages
ALTER SEQUENCE public.crm_stages_id_seq OWNED BY public.crm_stages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.crm_tags
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   color:                     integer                    default=0
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.crm_tags
CREATE TABLE public.crm_tags (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    color integer DEFAULT 0,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : crm_tags
CREATE SEQUENCE public.crm_tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : crm_tags
ALTER SEQUENCE public.crm_tags_id_seq OWNED BY public.crm_tags.id;

-- ==============================================================================
-- SECTION: documents  Documents / EDI (المستندات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 4
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.edi_certificates
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(256)               NOT NULL
--   cert_content:              bytea                      
--   private_key:               bytea                      
--   public_key:                bytea                      
--   csr:                       text                       
--   csid:                      varchar(512)               
--   secret:                    text                       
--   company_id:                bigint                     NOT NULL
--   is_production:             boolean                    default=false
--   expiration_date:           timestamptz                
--   active:                    boolean                    default=false
--   created_at:                timestamptz                default=now()
--   compliance_csid:           varchar(512)               
--   production_csid:           varchar(512)               
--   compliance_request_id:     varchar(128)               
--   onboarding_status:         varchar(32)                default='pending'
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.edi_certificates
CREATE TABLE public.edi_certificates (
    id bigint NOT NULL,
    name character varying(256) NOT NULL,
    cert_content bytea,
    private_key bytea,
    public_key bytea,
    csr text,
    csid character varying(512),
    secret text,
    company_id bigint NOT NULL,
    is_production boolean DEFAULT false NOT NULL,
    expiration_date timestamp with time zone,
    active boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    compliance_csid character varying(512),
    production_csid character varying(512),
    compliance_request_id character varying(128),
    onboarding_status character varying(32) DEFAULT 'pending'::character varying
);

-- CREATE SEQUENCE : edi_certificates
CREATE SEQUENCE public.edi_certificates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : edi_certificates
ALTER SEQUENCE public.edi_certificates_id_seq OWNED BY public.edi_certificates.id;

-- ----------------------------------------------------------------------
-- TABLE: public.edi_documents
--   id:                        bigint                     PK NOT NULL
--   move_id:                   bigint                     NOT NULL
--   format:                    varchar(32)                default='ubl_2_1'
--   transaction_type:          varchar(16)                default='standard'
--   state:                     varchar(32)                default='to_send'
--   xml_content:               bytea                      
--   hash:                      varchar(128)               
--   qr_code:                   varchar(512)               
--   error_msg:                 text                       
--   sent_at:                   timestamptz                
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   uuid:                      varchar(36)                
--   previous_hash:             varchar(128)               
--   signature:                 text                       
--   zatca_status:              varchar(32)                
--   zatca_request_id:          varchar(128)               
--   cleared_xml:               text                       
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.edi_documents
CREATE TABLE public.edi_documents (
    id bigint NOT NULL,
    move_id bigint NOT NULL,
    format character varying(32) DEFAULT 'ubl_2_1'::character varying NOT NULL,
    transaction_type character varying(16) DEFAULT 'standard'::character varying NOT NULL,
    state character varying(32) DEFAULT 'to_send'::character varying NOT NULL,
    xml_content bytea,
    hash character varying(128),
    qr_code character varying(512),
    error_msg text,
    sent_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    uuid character varying(36),
    previous_hash character varying(128),
    signature text,
    zatca_status character varying(32),
    zatca_request_id character varying(128),
    cleared_xml text
);

-- CREATE SEQUENCE : edi_documents
CREATE SEQUENCE public.edi_documents_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : edi_documents
ALTER SEQUENCE public.edi_documents_id_seq OWNED BY public.edi_documents.id;

-- ----------------------------------------------------------------------
-- TABLE: public.edi_zatca_submissions
--   id:                        bigint                     PK NOT NULL
--   edi_document_id:           bigint                     NOT NULL
--   submission_type:           varchar(16)                NOT NULL
--   request_body:              text                       NOT NULL
--   response_body:             text                       
--   response_status:           varchar(32)                
--   warnings:                  jsonb                      
--   errors:                    jsonb                      
--   submitted_at:              timestamptz                default=now()
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.edi_zatca_submissions
CREATE TABLE public.edi_zatca_submissions (
    id bigint NOT NULL,
    edi_document_id bigint NOT NULL,
    submission_type character varying(16) NOT NULL,
    request_body text NOT NULL,
    response_body text,
    response_status character varying(32),
    warnings jsonb,
    errors jsonb,
    submitted_at timestamp with time zone DEFAULT now() NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : edi_zatca_submissions
CREATE SEQUENCE public.edi_zatca_submissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : edi_zatca_submissions
ALTER SEQUENCE public.edi_zatca_submissions_id_seq OWNED BY public.edi_zatca_submissions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.ir_attachments
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   filename:                  varchar(255)               NOT NULL
--   mimetype:                  varchar(255)               default='application/octet-stream'
--   file_size:                 bigint                     default=0
--   checksum:                  varchar(40)                
--   storage_path:              varchar(500)               
--   res_model:                 varchar(100)               
--   res_id:                    bigint                     
--   description:               text                       
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                ir_attachments_file_size_check CHECK ((file_size >= 0)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ir_attachments
CREATE TABLE public.ir_attachments (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    filename character varying(255) NOT NULL,
    mimetype character varying(255) DEFAULT 'application/octet-stream'::character varying NOT NULL,
    file_size bigint DEFAULT 0 NOT NULL,
    checksum character varying(40),
    storage_path character varying(500),
    res_model character varying(100),
    res_id bigint,
    description text,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT ir_attachments_file_size_check CHECK ((file_size >= 0))
);

-- CREATE SEQUENCE : ir_attachments
CREATE SEQUENCE public.ir_attachments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ir_attachments
ALTER SEQUENCE public.ir_attachments_id_seq OWNED BY public.ir_attachments.id;

-- ==============================================================================
-- SECTION: partners   Partners / Companies (الشركاء والمنشآت)
-- ------------------------------------------------------------------------------
-- Tables in this section: 2
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.res_companies
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   partner_id:                bigint                     
--   currency_id:               bigint                     NOT NULL
--   phone:                     varchar(50)                
--   email:                     varchar(255)               
--   website:                   varchar(255)               
--   vat:                       varchar(100)               
--   street:                    varchar(255)               
--   street2:                   varchar(255)               
--   city:                      varchar(100)               
--   state:                     varchar(100)               
--   country:                   varchar(100)               
--   zip_code:                  varchar(20)                
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   lc_journal_id:             bigint                     
--   attendance_kiosk_mode:     varchar(20)                default='barcode_pin'
--   attendance_kiosk_delay:    integer                    default=10
--   overtime_company_threshold: integer                    default=0
--   auto_check_out_tolerance:  float8                     default=0
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_companies
CREATE TABLE public.res_companies (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    partner_id bigint,
    currency_id bigint NOT NULL,
    phone character varying(50),
    email character varying(255),
    website character varying(255),
    vat character varying(100),
    street character varying(255),
    street2 character varying(255),
    city character varying(100),
    state character varying(100),
    country character varying(100),
    zip_code character varying(20),
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    lc_journal_id bigint,
    attendance_kiosk_mode character varying(20) DEFAULT 'barcode_pin'::character varying,
    attendance_kiosk_delay integer DEFAULT 10,
    overtime_company_threshold integer DEFAULT 0,
    auto_check_out_tolerance double precision DEFAULT 0
);

-- CREATE SEQUENCE : res_companies
CREATE SEQUENCE public.res_companies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_companies
ALTER SEQUENCE public.res_companies_id_seq OWNED BY public.res_companies.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_partners
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   email:                     public.citext              
--   phone:                     varchar(50)                
--   mobile:                    varchar(50)                
--   type:                      varchar(20)                default='individual'
--   is_customer:               boolean                    default=true
--   is_supplier:               boolean                    default=false
--   vat_number:                varchar(50)                
--   website:                   varchar(255)               
--   company_id:                bigint                     
--   parent_id:                 bigint                     
--   street:                    varchar(255)               
--   street2:                   varchar(255)               
--   city:                      varchar(100)               
--   state:                     varchar(100)               
--   country:                   varchar(100)               
--   zip_code:                  varchar(20)                
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_partners
CREATE TABLE public.res_partners (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    email public.citext,
    phone character varying(50),
    mobile character varying(50),
    type character varying(20) DEFAULT 'individual'::character varying NOT NULL,
    is_customer boolean DEFAULT true NOT NULL,
    is_supplier boolean DEFAULT false NOT NULL,
    vat_number character varying(50),
    website character varying(255),
    company_id bigint,
    parent_id bigint,
    street character varying(255),
    street2 character varying(255),
    city character varying(100),
    state character varying(100),
    country character varying(100),
    zip_code character varying(20),
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : res_partners
CREATE SEQUENCE public.res_partners_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_partners
ALTER SEQUENCE public.res_partners_id_seq OWNED BY public.res_partners.id;

-- ==============================================================================
-- SECTION: hr         HR / Recruitment / Attendance (الموارد البشرية)
-- ------------------------------------------------------------------------------
-- Tables in this section: 25
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.appointment_bookings
--   id:                        bigint                     PK NOT NULL
--   appointment_type_id:       bigint                     NOT NULL
--   event_id:                  bigint                     NOT NULL
--   staff_id:                  bigint                     NOT NULL
--   customer_name:             varchar(128)               NOT NULL
--   customer_email:            varchar(128)               NOT NULL
--   customer_phone:            varchar(64)                
--   start_time:                timestamptz                NOT NULL
--   end_time:                  timestamptz                NOT NULL
--   notes:                     text                       
--   status:                    varchar(32)                default='confirmed'
--   created_at:                timestamptz                default=now()
--   CONSTRAINT:                appointment_bookings_check CHECK ((end_time > start_time)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.appointment_bookings
CREATE TABLE public.appointment_bookings (
    id bigint NOT NULL,
    appointment_type_id bigint NOT NULL,
    event_id bigint NOT NULL,
    staff_id bigint NOT NULL,
    customer_name character varying(128) NOT NULL,
    customer_email character varying(128) NOT NULL,
    customer_phone character varying(64),
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    notes text,
    status character varying(32) DEFAULT 'confirmed'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT appointment_bookings_check CHECK ((end_time > start_time))
);

-- CREATE SEQUENCE : appointment_bookings
CREATE SEQUENCE public.appointment_bookings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : appointment_bookings
ALTER SEQUENCE public.appointment_bookings_id_seq OWNED BY public.appointment_bookings.id;

-- ----------------------------------------------------------------------
-- TABLE: public.appointment_slots
--   id:                        bigint                     PK NOT NULL
--   appointment_type_id:       bigint                     NOT NULL
--   day_of_week:               smallint                   NOT NULL
--   hour_from:                 numeric(4,2)               NOT NULL
--   hour_to:                   numeric(4,2)               NOT NULL
--   CONSTRAINT:                appointment_slots_check CHECK ((hour_to > hour_from)) 
--   CONSTRAINT:                appointment_slots_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.appointment_slots
CREATE TABLE public.appointment_slots (
    id bigint NOT NULL,
    appointment_type_id bigint NOT NULL,
    day_of_week smallint NOT NULL,
    hour_from numeric(4,2) NOT NULL,
    hour_to numeric(4,2) NOT NULL,
    CONSTRAINT appointment_slots_check CHECK ((hour_to > hour_from)),
    CONSTRAINT appointment_slots_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6)))
);

-- CREATE SEQUENCE : appointment_slots
CREATE SEQUENCE public.appointment_slots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : appointment_slots
ALTER SEQUENCE public.appointment_slots_id_seq OWNED BY public.appointment_slots.id;

-- ----------------------------------------------------------------------
-- TABLE: public.appointment_type_users
--   appointment_type_id:       bigint                     PK NOT NULL
--   user_id:                   bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.appointment_type_users
CREATE TABLE public.appointment_type_users (
    appointment_type_id bigint NOT NULL,
    user_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.appointment_types
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   slug:                      varchar(64)                NOT NULL
--   duration_minutes:          integer                    default=30
--   min_schedule_hours:        integer                    default=2
--   max_schedule_days:         integer                    default=30
--   assignation_method:        varchar(32)                default='round_robin'
--   location:                  varchar(256)               
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.appointment_types
CREATE TABLE public.appointment_types (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    slug character varying(64) NOT NULL,
    duration_minutes integer DEFAULT 30 NOT NULL,
    min_schedule_hours integer DEFAULT 2 NOT NULL,
    max_schedule_days integer DEFAULT 30 NOT NULL,
    assignation_method character varying(32) DEFAULT 'round_robin'::character varying NOT NULL,
    location character varying(256),
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : appointment_types
CREATE SEQUENCE public.appointment_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : appointment_types
ALTER SEQUENCE public.appointment_types_id_seq OWNED BY public.appointment_types.id;

-- ----------------------------------------------------------------------
-- TABLE: public.calendar_attendees
--   id:                        bigint                     PK NOT NULL
--   event_id:                  bigint                     NOT NULL
--   partner_id:                bigint                     
--   email:                     varchar(128)               NOT NULL
--   name:                      varchar(128)               NOT NULL
--   status:                    varchar(32)                default='needs_action'
--   is_owner:                  boolean                    default=false
--   token:                     varchar(64)                NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.calendar_attendees
CREATE TABLE public.calendar_attendees (
    id bigint NOT NULL,
    event_id bigint NOT NULL,
    partner_id bigint,
    email character varying(128) NOT NULL,
    name character varying(128) NOT NULL,
    status character varying(32) DEFAULT 'needs_action'::character varying NOT NULL,
    is_owner boolean DEFAULT false NOT NULL,
    token character varying(64) NOT NULL
);

-- CREATE SEQUENCE : calendar_attendees
CREATE SEQUENCE public.calendar_attendees_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : calendar_attendees
ALTER SEQUENCE public.calendar_attendees_id_seq OWNED BY public.calendar_attendees.id;

-- ----------------------------------------------------------------------
-- TABLE: public.calendar_event_alarms
--   id:                        bigint                     PK NOT NULL
--   event_id:                  bigint                     NOT NULL
--   alarm_type:                varchar(32)                default='notification'
--   duration_minutes:          integer                    default=15
--   message:                   text                       
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.calendar_event_alarms
CREATE TABLE public.calendar_event_alarms (
    id bigint NOT NULL,
    event_id bigint NOT NULL,
    alarm_type character varying(32) DEFAULT 'notification'::character varying NOT NULL,
    duration_minutes integer DEFAULT 15 NOT NULL,
    message text
);

-- CREATE SEQUENCE : calendar_event_alarms
CREATE SEQUENCE public.calendar_event_alarms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : calendar_event_alarms
ALTER SEQUENCE public.calendar_event_alarms_id_seq OWNED BY public.calendar_event_alarms.id;

-- ----------------------------------------------------------------------
-- TABLE: public.calendar_events
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(256)               NOT NULL
--   description:               text                       
--   start_date:                timestamptz                NOT NULL
--   stop_date:                 timestamptz                NOT NULL
--   duration:                  numeric(6,2)               default=1.0
--   allday:                    boolean                    default=false
--   location:                  varchar(256)               
--   video_url:                 varchar(512)               
--   privacy:                   varchar(32)                default='public'
--   show_as:                   varchar(16)                default='busy'
--   user_id:                   bigint                     NOT NULL
--   res_model:                 varchar(64)                
--   res_id:                    bigint                     
--   recurrence_id:             bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                calendar_events_check CHECK ((stop_date > start_date)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.calendar_events
CREATE TABLE public.calendar_events (
    id bigint NOT NULL,
    name character varying(256) NOT NULL,
    description text,
    start_date timestamp with time zone NOT NULL,
    stop_date timestamp with time zone NOT NULL,
    duration numeric(6,2) DEFAULT 1.0 NOT NULL,
    allday boolean DEFAULT false NOT NULL,
    location character varying(256),
    video_url character varying(512),
    privacy character varying(32) DEFAULT 'public'::character varying NOT NULL,
    show_as character varying(16) DEFAULT 'busy'::character varying NOT NULL,
    user_id bigint NOT NULL,
    res_model character varying(64),
    res_id bigint,
    recurrence_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT calendar_events_check CHECK ((stop_date > start_date))
);

-- CREATE SEQUENCE : calendar_events
CREATE SEQUENCE public.calendar_events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : calendar_events
ALTER SEQUENCE public.calendar_events_id_seq OWNED BY public.calendar_events.id;

-- ----------------------------------------------------------------------
-- TABLE: public.calendar_recurrences
--   id:                        bigint                     PK NOT NULL
--   rrule:                     varchar(256)               NOT NULL
--   count:                     integer                    
--   until:                     timestamptz                
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.calendar_recurrences
CREATE TABLE public.calendar_recurrences (
    id bigint NOT NULL,
    rrule character varying(256) NOT NULL,
    count integer,
    until timestamp with time zone,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : calendar_recurrences
CREATE SEQUENCE public.calendar_recurrences_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : calendar_recurrences
ALTER SEQUENCE public.calendar_recurrences_id_seq OWNED BY public.calendar_recurrences.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_attendance
--   id:                        bigint                     PK NOT NULL
--   employee_id:               bigint                     NOT NULL
--   check_in:                  timestamptz                NOT NULL
--   check_out:                 timestamptz                
--   worked_hours:              float8                     default=0
--   expected_hours:            float8                     default=0
--   overtime_hours:            float8                     default=0
--   overtime_status:           varchar(20)                default='to_approve'
--   in_latitude:               float8                     
--   in_longitude:              float8                     
--   in_ip_address:             varchar(45)                
--   in_browser:                text                       
--   in_mode:                   varchar(20)                
--   out_latitude:              float8                     
--   out_longitude:             float8                     
--   out_ip_address:            varchar(45)                
--   out_browser:               text                       
--   out_mode:                  varchar(20)                
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_attendance
CREATE TABLE public.hr_attendance (
    id bigint NOT NULL,
    employee_id bigint NOT NULL,
    check_in timestamp with time zone NOT NULL,
    check_out timestamp with time zone,
    worked_hours double precision DEFAULT 0,
    expected_hours double precision DEFAULT 0,
    overtime_hours double precision DEFAULT 0,
    overtime_status character varying(20) DEFAULT 'to_approve'::character varying,
    in_latitude double precision,
    in_longitude double precision,
    in_ip_address character varying(45),
    in_browser text,
    in_mode character varying(20),
    out_latitude double precision,
    out_longitude double precision,
    out_ip_address character varying(45),
    out_browser text,
    out_mode character varying(20),
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : hr_attendance
CREATE SEQUENCE public.hr_attendance_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_attendance
ALTER SEQUENCE public.hr_attendance_id_seq OWNED BY public.hr_attendance.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_departments
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   complete_name:             jsonb                      
--   parent_id:                 bigint                     
--   manager_id:                bigint                     
--   company_id:                bigint                     
--   color:                     integer                    default=0
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_departments
CREATE TABLE public.hr_departments (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    complete_name jsonb,
    parent_id bigint,
    manager_id bigint,
    company_id bigint,
    color integer DEFAULT 0,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : hr_departments
CREATE SEQUENCE public.hr_departments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_departments
ALTER SEQUENCE public.hr_departments_id_seq OWNED BY public.hr_departments.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_employees
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   partner_id:                bigint                     
--   department_id:             bigint                     
--   job_id:                    bigint                     
--   job_title:                 varchar(255)               
--   manager_id:                bigint                     
--   work_email:                varchar(255)               
--   work_phone:                varchar(50)                
--   work_location:             varchar(255)               
--   hire_date:                 date                       
--   gender:                    varchar(20)                default='other'
--   marital_status:            varchar(20)                default='single'
--   identification_id:         varchar(100)               
--   bank_account_no:           varchar(100)               
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   overtime_employee_threshold: integer                    default=0
--   expense_manager_id:        bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_employees
CREATE TABLE public.hr_employees (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    partner_id bigint,
    department_id bigint,
    job_id bigint,
    job_title character varying(255),
    manager_id bigint,
    work_email character varying(255),
    work_phone character varying(50),
    work_location character varying(255),
    hire_date date,
    gender character varying(20) DEFAULT 'other'::character varying,
    marital_status character varying(20) DEFAULT 'single'::character varying,
    identification_id character varying(100),
    bank_account_no character varying(100),
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    overtime_employee_threshold integer DEFAULT 0,
    expense_manager_id bigint
);

-- CREATE SEQUENCE : hr_employees
CREATE SEQUENCE public.hr_employees_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_employees
ALTER SEQUENCE public.hr_employees_id_seq OWNED BY public.hr_employees.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_expense_taxes
--   expense_id:                bigint                     PK NOT NULL
--   tax_id:                    bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_expense_taxes
CREATE TABLE public.hr_expense_taxes (
    expense_id bigint NOT NULL,
    tax_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.hr_expenses
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   date:                      date                       default=CURRENT_DATE
--   employee_id:               bigint                     NOT NULL
--   manager_id:                bigint                     
--   department_id:             bigint                     
--   product_id:                bigint                     
--   unit_amount:               numeric(19,4)              default=0
--   quantity:                  numeric(19,4)              default=1
--   total_amount:              numeric(19,4)              default=0
--   untaxed_amount:            numeric(19,4)              default=0
--   tax_amount:                numeric(19,4)              default=0
--   currency_id:               bigint                     NOT NULL
--   payment_mode:              varchar(50)                default='own_account'
--   account_id:                bigint                     
--   analytic_account_id:       bigint                     
--   account_move_id:           bigint                     
--   vendor_id:                 bigint                     
--   description:               text                       
--   state:                     varchar(50)                default='draft'
--   approval_date:             timestamptz                
--   refuse_reason:             text                       
--   split_origin_id:           bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   attachment_checksums:      text[]                     default='{}'::text[]
--   CONSTRAINT:                hr_expenses_amounts_check CHECK (((unit_amount >= (0)::numeric) AND (total_amount >= (0)::numeric) AND (untaxed_amount >= (0)::numeric) AND (tax_amount >= (0)::numeric))) 
--   CONSTRAINT:                hr_expenses_payment_mode_check CHECK (((payment_mode)::text = ANY ((ARRAY['own_account'::varchar, 'company_account'::varchar])::text[]))) 
--   CONSTRAINT:                hr_expenses_quantity_check CHECK ((quantity > (0)::numeric)) 
--   CONSTRAINT:                hr_expenses_state_check CHECK (((state)::text = ANY ((ARRAY['draft'::varchar, 'submitted'::varchar, 'approved'::varchar, 'posted'::varchar, 'in_payment'::varchar, 'paid'::varchar, 'refused'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_expenses
CREATE TABLE public.hr_expenses (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    employee_id bigint NOT NULL,
    manager_id bigint,
    department_id bigint,
    product_id bigint,
    unit_amount numeric(19,4) DEFAULT 0 NOT NULL,
    quantity numeric(19,4) DEFAULT 1 NOT NULL,
    total_amount numeric(19,4) DEFAULT 0 NOT NULL,
    untaxed_amount numeric(19,4) DEFAULT 0 NOT NULL,
    tax_amount numeric(19,4) DEFAULT 0 NOT NULL,
    currency_id bigint NOT NULL,
    payment_mode character varying(50) DEFAULT 'own_account'::character varying NOT NULL,
    account_id bigint,
    analytic_account_id bigint,
    account_move_id bigint,
    vendor_id bigint,
    description text,
    state character varying(50) DEFAULT 'draft'::character varying NOT NULL,
    approval_date timestamp with time zone,
    refuse_reason text,
    split_origin_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    attachment_checksums text[] DEFAULT '{}'::text[] NOT NULL,
    CONSTRAINT hr_expenses_amounts_check CHECK (((unit_amount >= (0)::numeric) AND (total_amount >= (0)::numeric) AND (untaxed_amount >= (0)::numeric) AND (tax_amount >= (0)::numeric))),
    CONSTRAINT hr_expenses_payment_mode_check CHECK (((payment_mode)::text = ANY ((ARRAY['own_account'::character varying, 'company_account'::character varying])::text[]))),
    CONSTRAINT hr_expenses_quantity_check CHECK ((quantity > (0)::numeric)),
    CONSTRAINT hr_expenses_state_check CHECK (((state)::text = ANY ((ARRAY['draft'::character varying, 'submitted'::character varying, 'approved'::character varying, 'posted'::character varying, 'in_payment'::character varying, 'paid'::character varying, 'refused'::character varying])::text[])))
);

-- CREATE SEQUENCE : hr_expenses
CREATE SEQUENCE public.hr_expenses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_expenses
ALTER SEQUENCE public.hr_expenses_id_seq OWNED BY public.hr_expenses.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_jobs
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   department_id:             bigint                     
--   description:               text                       
--   expected_employees:        integer                    default=1
--   no_of_employee:            integer                    default=0
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                hr_jobs_expected_employees_check CHECK ((expected_employees >= 0)) 
--   CONSTRAINT:                hr_jobs_no_of_employee_check CHECK ((no_of_employee >= 0)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_jobs
CREATE TABLE public.hr_jobs (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    department_id bigint,
    description text,
    expected_employees integer DEFAULT 1 NOT NULL,
    no_of_employee integer DEFAULT 0 NOT NULL,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT hr_jobs_expected_employees_check CHECK ((expected_employees >= 0)),
    CONSTRAINT hr_jobs_no_of_employee_check CHECK ((no_of_employee >= 0))
);

-- CREATE SEQUENCE : hr_jobs
CREATE SEQUENCE public.hr_jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_jobs
ALTER SEQUENCE public.hr_jobs_id_seq OWNED BY public.hr_jobs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_leave_allocations
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   employee_id:               bigint                     NOT NULL
--   leave_type:                varchar(50)                default='annual'
--   allocated_days:            numeric(5,2)               NOT NULL
--   year:                      integer                    NOT NULL
--   state:                     varchar(20)                default='approved'
--   notes:                     text                       
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                hr_leave_allocations_allocated_days_check CHECK ((allocated_days >= (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_leave_allocations
CREATE TABLE public.hr_leave_allocations (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    employee_id bigint NOT NULL,
    leave_type character varying(50) DEFAULT 'annual'::character varying NOT NULL,
    allocated_days numeric(5,2) NOT NULL,
    year integer NOT NULL,
    state character varying(20) DEFAULT 'approved'::character varying NOT NULL,
    notes text,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT hr_leave_allocations_allocated_days_check CHECK ((allocated_days >= (0)::numeric))
);

-- CREATE SEQUENCE : hr_leave_allocations
CREATE SEQUENCE public.hr_leave_allocations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_leave_allocations
ALTER SEQUENCE public.hr_leave_allocations_id_seq OWNED BY public.hr_leave_allocations.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_leave_requests
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   employee_id:               bigint                     NOT NULL
--   leave_type:                varchar(50)                default='annual'
--   date_from:                 date                       NOT NULL
--   date_to:                   date                       NOT NULL
--   days:                      numeric(5,2)               NOT NULL
--   state:                     varchar(20)                default='draft'
--   description:               text                       
--   approver_id:               bigint                     
--   refusal_reason:            text                       
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                chk_hr_leave_dates CHECK ((date_to >= date_from)) 
--   CONSTRAINT:                hr_leave_requests_days_check CHECK ((days > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_leave_requests
CREATE TABLE public.hr_leave_requests (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    employee_id bigint NOT NULL,
    leave_type character varying(50) DEFAULT 'annual'::character varying NOT NULL,
    date_from date NOT NULL,
    date_to date NOT NULL,
    days numeric(5,2) NOT NULL,
    state character varying(20) DEFAULT 'draft'::character varying NOT NULL,
    description text,
    approver_id bigint,
    refusal_reason text,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT chk_hr_leave_dates CHECK ((date_to >= date_from)),
    CONSTRAINT hr_leave_requests_days_check CHECK ((days > (0)::numeric))
);

-- CREATE SEQUENCE : hr_leave_requests
CREATE SEQUENCE public.hr_leave_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_leave_requests
ALTER SEQUENCE public.hr_leave_requests_id_seq OWNED BY public.hr_leave_requests.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_overtime_lines
--   id:                        bigint                     PK NOT NULL
--   employee_id:               bigint                     NOT NULL
--   attendance_id:             bigint                     
--   date:                      date                       NOT NULL
--   duration:                  float8                     default=0
--   manual_duration:           float8                     default=0
--   status:                    varchar(20)                default='to_approve'
--   time_start:                timestamptz                
--   time_stop:                 timestamptz                
--   rule_ids:                  bigint[]                   
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_overtime_lines
CREATE TABLE public.hr_overtime_lines (
    id bigint NOT NULL,
    employee_id bigint NOT NULL,
    attendance_id bigint,
    date date NOT NULL,
    duration double precision DEFAULT 0,
    manual_duration double precision DEFAULT 0,
    status character varying(20) DEFAULT 'to_approve'::character varying,
    time_start timestamp with time zone,
    time_stop timestamp with time zone,
    rule_ids bigint[],
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : hr_overtime_lines
CREATE SEQUENCE public.hr_overtime_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_overtime_lines
ALTER SEQUENCE public.hr_overtime_lines_id_seq OWNED BY public.hr_overtime_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_overtime_rules
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   base_off:                  varchar(20)                NOT NULL
--   timing_type:               varchar(20)                
--   timing_start:              float8                     
--   multiplier:                float8                     default=1.0
--   active:                    boolean                    default=true
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_overtime_rules
CREATE TABLE public.hr_overtime_rules (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    base_off character varying(20) NOT NULL,
    timing_type character varying(20),
    timing_start double precision,
    multiplier double precision DEFAULT 1.0,
    active boolean DEFAULT true,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : hr_overtime_rules
CREATE SEQUENCE public.hr_overtime_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_overtime_rules
ALTER SEQUENCE public.hr_overtime_rules_id_seq OWNED BY public.hr_overtime_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.hr_work_entries
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   employee_id:               bigint                     NOT NULL
--   work_entry_type:           varchar(32)                default='attendance'
--   date_start:                timestamptz                NOT NULL
--   date_stop:                 timestamptz                NOT NULL
--   duration_hours:            numeric(6,2)               NOT NULL
--   state:                     varchar(32)                default='draft'
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                hr_work_entries_check CHECK ((date_stop > date_start)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.hr_work_entries
CREATE TABLE public.hr_work_entries (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    employee_id bigint NOT NULL,
    work_entry_type character varying(32) DEFAULT 'attendance'::character varying NOT NULL,
    date_start timestamp with time zone NOT NULL,
    date_stop timestamp with time zone NOT NULL,
    duration_hours numeric(6,2) NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT hr_work_entries_check CHECK ((date_stop > date_start))
);

-- CREATE SEQUENCE : hr_work_entries
CREATE SEQUENCE public.hr_work_entries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : hr_work_entries
ALTER SEQUENCE public.hr_work_entries_id_seq OWNED BY public.hr_work_entries.id;

-- ----------------------------------------------------------------------
-- TABLE: public.planning_roles
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   color:                     varchar(16)                default='#3B82F6'
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.planning_roles
CREATE TABLE public.planning_roles (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    color character varying(16) DEFAULT '#3B82F6'::character varying,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : planning_roles
CREATE SEQUENCE public.planning_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : planning_roles
ALTER SEQUENCE public.planning_roles_id_seq OWNED BY public.planning_roles.id;

-- ----------------------------------------------------------------------
-- TABLE: public.planning_shifts
--   id:                        bigint                     PK NOT NULL
--   employee_id:               bigint                     
--   role_id:                   bigint                     NOT NULL
--   start_at:                  timestamptz                NOT NULL
--   end_at:                    timestamptz                NOT NULL
--   allocated_hours:           numeric(6,2)               NOT NULL
--   is_published:              boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.planning_shifts
CREATE TABLE public.planning_shifts (
    id bigint NOT NULL,
    employee_id bigint,
    role_id bigint NOT NULL,
    start_at timestamp with time zone NOT NULL,
    end_at timestamp with time zone NOT NULL,
    allocated_hours numeric(6,2) NOT NULL,
    is_published boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : planning_shifts
CREATE SEQUENCE public.planning_shifts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : planning_shifts
ALTER SEQUENCE public.planning_shifts_id_seq OWNED BY public.planning_shifts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.recruitment_applicants
--   id:                        bigint                     PK NOT NULL
--   partner_name:              varchar(128)               NOT NULL
--   email:                     varchar(128)               NOT NULL
--   phone:                     varchar(32)                
--   job_id:                    bigint                     NOT NULL
--   department_id:             bigint                     
--   stage_id:                  bigint                     NOT NULL
--   recruiter_user_id:         bigint                     
--   priority:                  integer                    default=0
--   salary_expected:           numeric(15,2)              default=0
--   salary_proposed:           numeric(15,2)              default=0
--   availability:              date                       
--   refusal_reason:            text                       
--   resume_url:                text                       
--   employee_id:               bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.recruitment_applicants
CREATE TABLE public.recruitment_applicants (
    id bigint NOT NULL,
    partner_name character varying(128) NOT NULL,
    email character varying(128) NOT NULL,
    phone character varying(32),
    job_id bigint NOT NULL,
    department_id bigint,
    stage_id bigint NOT NULL,
    recruiter_user_id bigint,
    priority integer DEFAULT 0 NOT NULL,
    salary_expected numeric(15,2) DEFAULT 0 NOT NULL,
    salary_proposed numeric(15,2) DEFAULT 0 NOT NULL,
    availability date,
    refusal_reason text,
    resume_url text,
    employee_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : recruitment_applicants
CREATE SEQUENCE public.recruitment_applicants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : recruitment_applicants
ALTER SEQUENCE public.recruitment_applicants_id_seq OWNED BY public.recruitment_applicants.id;

-- ----------------------------------------------------------------------
-- TABLE: public.recruitment_interviews
--   id:                        bigint                     PK NOT NULL
--   applicant_id:              bigint                     NOT NULL
--   interviewer_id:            bigint                     NOT NULL
--   event_id:                  bigint                     
--   interview_date:            timestamptz                NOT NULL
--   score:                     integer                    default=0
--   feedback:                  text                       
--   recommendation:            varchar(32)                default='consider'
--   CONSTRAINT:                recruitment_interviews_score_check CHECK (((score >= 0) AND (score <= 10))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.recruitment_interviews
CREATE TABLE public.recruitment_interviews (
    id bigint NOT NULL,
    applicant_id bigint NOT NULL,
    interviewer_id bigint NOT NULL,
    event_id bigint,
    interview_date timestamp with time zone NOT NULL,
    score integer DEFAULT 0 NOT NULL,
    feedback text,
    recommendation character varying(32) DEFAULT 'consider'::character varying NOT NULL,
    CONSTRAINT recruitment_interviews_score_check CHECK (((score >= 0) AND (score <= 10)))
);

-- CREATE SEQUENCE : recruitment_interviews
CREATE SEQUENCE public.recruitment_interviews_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : recruitment_interviews
ALTER SEQUENCE public.recruitment_interviews_id_seq OWNED BY public.recruitment_interviews.id;

-- ----------------------------------------------------------------------
-- TABLE: public.recruitment_stages
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   sequence:                  integer                    default=10
--   folded:                    boolean                    default=false
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.recruitment_stages
CREATE TABLE public.recruitment_stages (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    folded boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : recruitment_stages
CREATE SEQUENCE public.recruitment_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : recruitment_stages
ALTER SEQUENCE public.recruitment_stages_id_seq OWNED BY public.recruitment_stages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.resource_calendars
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   hours_per_day:             numeric(4,2)               default=8.0
--   full_time_required_hours:  numeric(4,2)               default=40.0
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.resource_calendars
CREATE TABLE public.resource_calendars (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    hours_per_day numeric(4,2) DEFAULT 8.0 NOT NULL,
    full_time_required_hours numeric(4,2) DEFAULT 40.0 NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : resource_calendars
CREATE SEQUENCE public.resource_calendars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : resource_calendars
ALTER SEQUENCE public.resource_calendars_id_seq OWNED BY public.resource_calendars.id;

-- ==============================================================================
-- SECTION: mrp        MRP - Manufacturing (تخطيط موارد التصنيع)
-- ------------------------------------------------------------------------------
-- Tables in this section: 16
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_bom_lines
--   id:                        bigint                     PK NOT NULL
--   bom_id:                    bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   quantity:                  float8                     default=1.0
--   uom_id:                    bigint                     NOT NULL
--   operation_id:              bigint                     
--   sequence:                  integer                    default=10
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_bom_lines
CREATE TABLE public.mrp_bom_lines (
    id bigint NOT NULL,
    bom_id bigint NOT NULL,
    product_id bigint NOT NULL,
    quantity double precision DEFAULT 1.0 NOT NULL,
    uom_id bigint NOT NULL,
    operation_id bigint,
    sequence integer DEFAULT 10
);

-- CREATE SEQUENCE : mrp_bom_lines
CREATE SEQUENCE public.mrp_bom_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_bom_lines
ALTER SEQUENCE public.mrp_bom_lines_id_seq OWNED BY public.mrp_bom_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_boms
--   id:                        bigint                     PK NOT NULL
--   code:                      varchar(64)                
--   product_id:                bigint                     NOT NULL
--   product_qty:               float8                     default=1.0
--   uom_id:                    bigint                     NOT NULL
--   type:                      varchar(32)                default='normal'
--   ready_to_produce:          varchar(32)                default='all_available'
--   consumption:               varchar(32)                default='flexible'
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_boms
CREATE TABLE public.mrp_boms (
    id bigint NOT NULL,
    code character varying(64),
    product_id bigint NOT NULL,
    product_qty double precision DEFAULT 1.0 NOT NULL,
    uom_id bigint NOT NULL,
    type character varying(32) DEFAULT 'normal'::character varying NOT NULL,
    ready_to_produce character varying(32) DEFAULT 'all_available'::character varying,
    consumption character varying(32) DEFAULT 'flexible'::character varying,
    active boolean DEFAULT true,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : mrp_boms
CREATE SEQUENCE public.mrp_boms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_boms
ALTER SEQUENCE public.mrp_boms_id_seq OWNED BY public.mrp_boms.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_capacity_slots
--   id:                        bigint                     PK NOT NULL
--   workcenter_id:             bigint                     NOT NULL
--   date_start:                timestamptz                NOT NULL
--   date_end:                  timestamptz                NOT NULL
--   available_hours:           numeric(10,2)              default=0
--   allocated_hours:           numeric(10,2)              default=0
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                mrp_capacity_slots_check CHECK ((date_end > date_start)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_capacity_slots
CREATE TABLE public.mrp_capacity_slots (
    id bigint NOT NULL,
    workcenter_id bigint NOT NULL,
    date_start timestamp with time zone NOT NULL,
    date_end timestamp with time zone NOT NULL,
    available_hours numeric(10,2) DEFAULT 0 NOT NULL,
    allocated_hours numeric(10,2) DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT mrp_capacity_slots_check CHECK ((date_end > date_start))
);

-- CREATE SEQUENCE : mrp_capacity_slots
CREATE SEQUENCE public.mrp_capacity_slots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_capacity_slots
ALTER SEQUENCE public.mrp_capacity_slots_id_seq OWNED BY public.mrp_capacity_slots.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_productions
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   priority:                  integer                    default=0
--   backorder_sequence:        integer                    default=0
--   origin:                    varchar(255)               
--   product_id:                bigint                     NOT NULL
--   product_qty:               float8                     default=1.0
--   uom_id:                    bigint                     NOT NULL
--   qty_producing:             float8                     default=0
--   qty_produced:              float8                     default=0
--   bom_id:                    bigint                     NOT NULL
--   picking_type_id:           bigint                     NOT NULL
--   location_src_id:           bigint                     NOT NULL
--   location_dest_id:          bigint                     NOT NULL
--   date_deadline:             timestamptz                
--   date_start:                timestamptz                NOT NULL
--   date_finished:             timestamptz                
--   state:                     varchar(32)                default='draft'
--   reservation_state:         varchar(32)                default='confirmed'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_productions
CREATE TABLE public.mrp_productions (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    priority integer DEFAULT 0,
    backorder_sequence integer DEFAULT 0,
    origin character varying(255),
    product_id bigint NOT NULL,
    product_qty double precision DEFAULT 1.0 NOT NULL,
    uom_id bigint NOT NULL,
    qty_producing double precision DEFAULT 0,
    qty_produced double precision DEFAULT 0,
    bom_id bigint NOT NULL,
    picking_type_id bigint NOT NULL,
    location_src_id bigint NOT NULL,
    location_dest_id bigint NOT NULL,
    date_deadline timestamp with time zone,
    date_start timestamp with time zone NOT NULL,
    date_finished timestamp with time zone,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    reservation_state character varying(32) DEFAULT 'confirmed'::character varying,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : mrp_productions
CREATE SEQUENCE public.mrp_productions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_productions
ALTER SEQUENCE public.mrp_productions_id_seq OWNED BY public.mrp_productions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_productivity_losses
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   loss_type:                 varchar(32)                NOT NULL
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_productivity_losses
CREATE TABLE public.mrp_productivity_losses (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    loss_type character varying(32) NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : mrp_productivity_losses
CREATE SEQUENCE public.mrp_productivity_losses_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_productivity_losses
ALTER SEQUENCE public.mrp_productivity_losses_id_seq OWNED BY public.mrp_productivity_losses.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_quality_checks
--   id:                        bigint                     PK NOT NULL
--   point_id:                  bigint                     NOT NULL
--   workorder_id:              bigint                     
--   production_id:             bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   result:                    text                       default=''
--   measure_value:             float8                     
--   note:                      text                       
--   state:                     varchar(16)                default='none'
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_quality_checks
CREATE TABLE public.mrp_quality_checks (
    id bigint NOT NULL,
    point_id bigint NOT NULL,
    workorder_id bigint,
    production_id bigint NOT NULL,
    product_id bigint NOT NULL,
    result text DEFAULT ''::text NOT NULL,
    measure_value double precision,
    note text,
    state character varying(16) DEFAULT 'none'::character varying NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : mrp_quality_checks
CREATE SEQUENCE public.mrp_quality_checks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_quality_checks
ALTER SEQUENCE public.mrp_quality_checks_id_seq OWNED BY public.mrp_quality_checks.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_quality_points
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   product_id:                bigint                     
--   operation_id:              bigint                     
--   workcenter_id:             bigint                     
--   check_type:                varchar(32)                NOT NULL
--   norm_min:                  float8                     
--   norm_max:                  float8                     
--   instructions:              text                       
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_quality_points
CREATE TABLE public.mrp_quality_points (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    product_id bigint,
    operation_id bigint,
    workcenter_id bigint,
    check_type character varying(32) NOT NULL,
    norm_min double precision,
    norm_max double precision,
    instructions text,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : mrp_quality_points
CREATE SEQUENCE public.mrp_quality_points_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_quality_points
ALTER SEQUENCE public.mrp_quality_points_id_seq OWNED BY public.mrp_quality_points.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_routing_operations
--   id:                        bigint                     PK NOT NULL
--   bom_id:                    bigint                     NOT NULL
--   workcenter_id:             bigint                     NOT NULL
--   name:                      varchar(255)               NOT NULL
--   sequence:                  integer                    default=10
--   time_mode:                 varchar(32)                default='manual'
--   time_cycle_manual:         float8                     default=0
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_routing_operations
CREATE TABLE public.mrp_routing_operations (
    id bigint NOT NULL,
    bom_id bigint NOT NULL,
    workcenter_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    sequence integer DEFAULT 10,
    time_mode character varying(32) DEFAULT 'manual'::character varying,
    time_cycle_manual double precision DEFAULT 0
);

-- CREATE SEQUENCE : mrp_routing_operations
CREATE SEQUENCE public.mrp_routing_operations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_routing_operations
ALTER SEQUENCE public.mrp_routing_operations_id_seq OWNED BY public.mrp_routing_operations.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_subcontracting_bom
--   id:                        bigint                     PK NOT NULL
--   bom_id:                    bigint                     NOT NULL
--   subcontractor_id:          bigint                     NOT NULL
--   lead_time_days:            integer                    default=0
--   cost_per_unit:             numeric(15,4)              default=0
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_subcontracting_bom
CREATE TABLE public.mrp_subcontracting_bom (
    id bigint NOT NULL,
    bom_id bigint NOT NULL,
    subcontractor_id bigint NOT NULL,
    lead_time_days integer DEFAULT 0 NOT NULL,
    cost_per_unit numeric(15,4) DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : mrp_subcontracting_bom
CREATE SEQUENCE public.mrp_subcontracting_bom_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_subcontracting_bom
ALTER SEQUENCE public.mrp_subcontracting_bom_id_seq OWNED BY public.mrp_subcontracting_bom.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_subcontracting_orders
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   production_id:             bigint                     NOT NULL
--   subcontractor_id:          bigint                     NOT NULL
--   purchase_order_id:         bigint                     
--   picking_out_id:            bigint                     
--   picking_in_id:             bigint                     
--   state:                     varchar(32)                default='draft'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_subcontracting_orders
CREATE TABLE public.mrp_subcontracting_orders (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    production_id bigint NOT NULL,
    subcontractor_id bigint NOT NULL,
    purchase_order_id bigint,
    picking_out_id bigint,
    picking_in_id bigint,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : mrp_subcontracting_orders
CREATE SEQUENCE public.mrp_subcontracting_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_subcontracting_orders
ALTER SEQUENCE public.mrp_subcontracting_orders_id_seq OWNED BY public.mrp_subcontracting_orders.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_unbuilds
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   product_id:                bigint                     NOT NULL
--   bom_id:                    bigint                     NOT NULL
--   mo_id:                     bigint                     
--   quantity:                  float8                     default=1.0
--   uom_id:                    bigint                     NOT NULL
--   location_id:               bigint                     NOT NULL
--   dest_location_id:          bigint                     NOT NULL
--   state:                     varchar(32)                default='draft'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_unbuilds
CREATE TABLE public.mrp_unbuilds (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    product_id bigint NOT NULL,
    bom_id bigint NOT NULL,
    mo_id bigint,
    quantity double precision DEFAULT 1.0 NOT NULL,
    uom_id bigint NOT NULL,
    location_id bigint NOT NULL,
    dest_location_id bigint NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : mrp_unbuilds
CREATE SEQUENCE public.mrp_unbuilds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_unbuilds
ALTER SEQUENCE public.mrp_unbuilds_id_seq OWNED BY public.mrp_unbuilds.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_workcenter_calendars
--   id:                        bigint                     PK NOT NULL
--   workcenter_id:             bigint                     NOT NULL
--   day_of_week:               smallint                   NOT NULL
--   hour_from:                 numeric(4,2)               NOT NULL
--   hour_to:                   numeric(4,2)               NOT NULL
--   attendance_type:           varchar(32)                default='working'
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                mrp_workcenter_calendars_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_workcenter_calendars
CREATE TABLE public.mrp_workcenter_calendars (
    id bigint NOT NULL,
    workcenter_id bigint NOT NULL,
    day_of_week smallint NOT NULL,
    hour_from numeric(4,2) NOT NULL,
    hour_to numeric(4,2) NOT NULL,
    attendance_type character varying(32) DEFAULT 'working'::character varying NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT mrp_workcenter_calendars_day_of_week_check CHECK (((day_of_week >= 0) AND (day_of_week <= 6)))
);

-- CREATE SEQUENCE : mrp_workcenter_calendars
CREATE SEQUENCE public.mrp_workcenter_calendars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_workcenter_calendars
ALTER SEQUENCE public.mrp_workcenter_calendars_id_seq OWNED BY public.mrp_workcenter_calendars.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_workcenter_productivity
--   id:                        bigint                     PK NOT NULL
--   workcenter_id:             bigint                     NOT NULL
--   workorder_id:              bigint                     
--   loss_id:                   bigint                     NOT NULL
--   loss_type:                 varchar(32)                NOT NULL
--   date_start:                timestamptz                NOT NULL
--   date_end:                  timestamptz                
--   duration:                  float8                     default=0
--   description:               text                       
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_workcenter_productivity
CREATE TABLE public.mrp_workcenter_productivity (
    id bigint NOT NULL,
    workcenter_id bigint NOT NULL,
    workorder_id bigint,
    loss_id bigint NOT NULL,
    loss_type character varying(32) NOT NULL,
    date_start timestamp with time zone NOT NULL,
    date_end timestamp with time zone,
    duration double precision DEFAULT 0 NOT NULL,
    description text,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : mrp_workcenter_productivity
CREATE SEQUENCE public.mrp_workcenter_productivity_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_workcenter_productivity
ALTER SEQUENCE public.mrp_workcenter_productivity_id_seq OWNED BY public.mrp_workcenter_productivity.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_workcenters
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   code:                      varchar(32)                
--   active:                    boolean                    default=true
--   sequence:                  integer                    default=10
--   company_id:                bigint                     NOT NULL
--   time_start:                float8                     default=0
--   time_stop:                 float8                     default=0
--   time_efficiency:           float8                     default=100.0
--   capacity:                  float8                     default=1.0
--   cost_per_hour:             float8                     default=0
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_workcenters
CREATE TABLE public.mrp_workcenters (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    code character varying(32),
    active boolean DEFAULT true,
    sequence integer DEFAULT 10,
    company_id bigint NOT NULL,
    time_start double precision DEFAULT 0,
    time_stop double precision DEFAULT 0,
    time_efficiency double precision DEFAULT 100.0,
    capacity double precision DEFAULT 1.0,
    cost_per_hour double precision DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : mrp_workcenters
CREATE SEQUENCE public.mrp_workcenters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_workcenters
ALTER SEQUENCE public.mrp_workcenters_id_seq OWNED BY public.mrp_workcenters.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_workorder_time_logs
--   id:                        bigint                     PK NOT NULL
--   workorder_id:              bigint                     NOT NULL
--   user_id:                   bigint                     default=0
--   date_start:                timestamptz                NOT NULL
--   date_end:                  timestamptz                
--   duration:                  float8                     default=0
--   loss_id:                   bigint                     
--   CONSTRAINT:                mrp_time_log_dates CHECK (((date_end IS NULL) OR (date_end >= date_start))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_workorder_time_logs
CREATE TABLE public.mrp_workorder_time_logs (
    id bigint NOT NULL,
    workorder_id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    date_start timestamp with time zone NOT NULL,
    date_end timestamp with time zone,
    duration double precision DEFAULT 0 NOT NULL,
    loss_id bigint,
    CONSTRAINT mrp_time_log_dates CHECK (((date_end IS NULL) OR (date_end >= date_start)))
);

-- CREATE SEQUENCE : mrp_workorder_time_logs
CREATE SEQUENCE public.mrp_workorder_time_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_workorder_time_logs
ALTER SEQUENCE public.mrp_workorder_time_logs_id_seq OWNED BY public.mrp_workorder_time_logs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mrp_workorders
--   id:                        bigint                     PK NOT NULL
--   production_id:             bigint                     NOT NULL
--   workcenter_id:             bigint                     NOT NULL
--   operation_id:              bigint                     NOT NULL
--   name:                      varchar(255)               NOT NULL
--   sequence:                  integer                    default=10
--   state:                     varchar(32)                default='ready'
--   duration_expected:         float8                     default=0
--   duration:                  float8                     default=0
--   date_start:                timestamptz                
--   date_finished:             timestamptz                
--   created_at:                timestamptz                default=CURRENT_TIMESTAMP
--   updated_at:                timestamptz                default=CURRENT_TIMESTAMP
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mrp_workorders
CREATE TABLE public.mrp_workorders (
    id bigint NOT NULL,
    production_id bigint NOT NULL,
    workcenter_id bigint NOT NULL,
    operation_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    sequence integer DEFAULT 10,
    state character varying(32) DEFAULT 'ready'::character varying NOT NULL,
    duration_expected double precision DEFAULT 0,
    duration double precision DEFAULT 0,
    date_start timestamp with time zone,
    date_finished timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : mrp_workorders
CREATE SEQUENCE public.mrp_workorders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mrp_workorders
ALTER SEQUENCE public.mrp_workorders_id_seq OWNED BY public.mrp_workorders.id;

-- ==============================================================================
-- SECTION: users      Users / Security / RBAC (المستخدمون والصلاحيات)
-- ------------------------------------------------------------------------------
-- Tables in this section: 7
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.portal_users
--   id:                        bigint                     PK NOT NULL
--   partner_id:                bigint                     NOT NULL
--   email:                     varchar(128)               NOT NULL
--   password_hash:             varchar(256)               NOT NULL
--   is_active:                 boolean                    default=true
--   last_login_at:             timestamptz                
--   company_id:                bigint                     NOT NULL
--   invite_token:              varchar(128)               
--   invite_accepted:           boolean                    default=false
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.portal_users
CREATE TABLE public.portal_users (
    id bigint NOT NULL,
    partner_id bigint NOT NULL,
    email character varying(128) NOT NULL,
    password_hash character varying(256) NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    last_login_at timestamp with time zone,
    company_id bigint NOT NULL,
    invite_token character varying(128),
    invite_accepted boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : portal_users
CREATE SEQUENCE public.portal_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : portal_users
ALTER SEQUENCE public.portal_users_id_seq OWNED BY public.portal_users.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_group_permissions
--   id:                        bigint                     PK NOT NULL
--   group_id:                  bigint                     NOT NULL
--   model:                     varchar(100)               NOT NULL
--   can_read:                  boolean                    default=false
--   can_create:                boolean                    default=false
--   can_update:                boolean                    default=false
--   can_delete:                boolean                    default=false
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_group_permissions
CREATE TABLE public.res_group_permissions (
    id bigint NOT NULL,
    group_id bigint NOT NULL,
    model character varying(100) NOT NULL,
    can_read boolean DEFAULT false NOT NULL,
    can_create boolean DEFAULT false NOT NULL,
    can_update boolean DEFAULT false NOT NULL,
    can_delete boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : res_group_permissions
CREATE SEQUENCE public.res_group_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_group_permissions
ALTER SEQUENCE public.res_group_permissions_id_seq OWNED BY public.res_group_permissions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_groups
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   category:                  varchar(100)               
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_groups
CREATE TABLE public.res_groups (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    category character varying(100),
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : res_groups
CREATE SEQUENCE public.res_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_groups
ALTER SEQUENCE public.res_groups_id_seq OWNED BY public.res_groups.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_groups_implied_rel
--   group_id:                  bigint                     PK NOT NULL
--   implied_group_id:          bigint                     PK NOT NULL
--   CONSTRAINT:                chk_res_groups_implied_not_self CHECK ((group_id <> implied_group_id)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_groups_implied_rel
CREATE TABLE public.res_groups_implied_rel (
    group_id bigint NOT NULL,
    implied_group_id bigint NOT NULL,
    CONSTRAINT chk_res_groups_implied_not_self CHECK ((group_id <> implied_group_id))
);

-- ----------------------------------------------------------------------
-- TABLE: public.res_groups_users_rel
--   group_id:                  bigint                     PK NOT NULL
--   user_id:                   bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_groups_users_rel
CREATE TABLE public.res_groups_users_rel (
    group_id bigint NOT NULL,
    user_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.res_record_rules
--   id:                        bigint                     PK NOT NULL
--   model:                     varchar(100)               NOT NULL
--   group_id:                  bigint                     
--   domain:                    jsonb                      NOT NULL
--   can_read:                  boolean                    default=false
--   can_create:                boolean                    default=false
--   can_update:                boolean                    default=false
--   can_delete:                boolean                    default=false
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_record_rules
CREATE TABLE public.res_record_rules (
    id bigint NOT NULL,
    model character varying(100) NOT NULL,
    group_id bigint,
    domain jsonb NOT NULL,
    can_read boolean DEFAULT false NOT NULL,
    can_create boolean DEFAULT false NOT NULL,
    can_update boolean DEFAULT false NOT NULL,
    can_delete boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : res_record_rules
CREATE SEQUENCE public.res_record_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_record_rules
ALTER SEQUENCE public.res_record_rules_id_seq OWNED BY public.res_record_rules.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_users
--   id:                        bigint                     PK NOT NULL
--   login:                     varchar(255)               NOT NULL
--   email:                     varchar(255)               
--   name:                      varchar(255)               NOT NULL
--   password_hash:             varchar(255)               NOT NULL
--   partner_id:                bigint                     NOT NULL
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
--   is_superuser:              boolean                    default=false
--   last_login_at:             timestamptz                
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   email_notifications_enabled: boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_users
CREATE TABLE public.res_users (
    id bigint NOT NULL,
    login character varying(255) NOT NULL,
    email character varying(255),
    name character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    partner_id bigint NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    is_superuser boolean DEFAULT false NOT NULL,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    email_notifications_enabled boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : res_users
CREATE SEQUENCE public.res_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_users
ALTER SEQUENCE public.res_users_id_seq OWNED BY public.res_users.id;

-- ==============================================================================
-- SECTION: analytics  Analytics / Analytic Accounting (التحليل)
-- ------------------------------------------------------------------------------
-- Tables in this section: 5
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.account_analytic_account
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   code:                      text                       
--   plan_id:                   bigint                     NOT NULL
--   root_plan_id:              bigint                     
--   partner_id:                bigint                     
--   color:                     integer                    
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_analytic_account
CREATE TABLE public.account_analytic_account (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    code text,
    plan_id bigint NOT NULL,
    root_plan_id bigint,
    partner_id bigint,
    color integer,
    company_id bigint,
    active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_analytic_account
CREATE SEQUENCE public.account_analytic_account_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_analytic_account
ALTER SEQUENCE public.account_analytic_account_id_seq OWNED BY public.account_analytic_account.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_analytic_applicability
--   id:                        bigint                     PK NOT NULL
--   analytic_plan_id:          bigint                     NOT NULL
--   business_domain:           text                       default='general'
--   applicability:             text                       NOT NULL
--   company_id:                bigint                     
--   sequence:                  integer                    default=10
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_analytic_applicability
CREATE TABLE public.account_analytic_applicability (
    id bigint NOT NULL,
    analytic_plan_id bigint NOT NULL,
    business_domain text DEFAULT 'general'::text NOT NULL,
    applicability text NOT NULL,
    company_id bigint,
    sequence integer DEFAULT 10,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_analytic_applicability
CREATE SEQUENCE public.account_analytic_applicability_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_analytic_applicability
ALTER SEQUENCE public.account_analytic_applicability_id_seq OWNED BY public.account_analytic_applicability.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_analytic_distribution_model
--   id:                        bigint                     PK NOT NULL
--   sequence:                  integer                    default=10
--   partner_id:                bigint                     
--   partner_category_id:       bigint                     
--   company_id:                bigint                     
--   analytic_distribution:     jsonb CONSTRAINT account_analytic_distribution_mo_analytic_distribution_not_null NOT NULL
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_analytic_distribution_model
CREATE TABLE public.account_analytic_distribution_model (
    id bigint NOT NULL,
    sequence integer DEFAULT 10,
    partner_id bigint,
    partner_category_id bigint,
    company_id bigint,
    analytic_distribution jsonb CONSTRAINT account_analytic_distribution_mo_analytic_distribution_not_null NOT NULL,
    active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_analytic_distribution_model
CREATE SEQUENCE public.account_analytic_distribution_model_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_analytic_distribution_model
ALTER SEQUENCE public.account_analytic_distribution_model_id_seq OWNED BY public.account_analytic_distribution_model.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_analytic_line
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   date:                      date                       NOT NULL
--   amount:                    float8                     default=0
--   unit_amount:               float8                     default=0
--   product_uom_id:            bigint                     
--   partner_id:                bigint                     
--   user_id:                   bigint                     NOT NULL
--   company_id:                bigint                     NOT NULL
--   currency_code:             text                       default='USD'
--   category:                  text                       default='other'
--   account_id:                bigint                     NOT NULL
--   analytic_distribution:     jsonb                      
--   move_line_id:              bigint                     
--   general_account_id:        bigint                     
--   source:                    text                       default='manual'
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_analytic_line
CREATE TABLE public.account_analytic_line (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    date date NOT NULL,
    amount double precision DEFAULT 0 NOT NULL,
    unit_amount double precision DEFAULT 0 NOT NULL,
    product_uom_id bigint,
    partner_id bigint,
    user_id bigint NOT NULL,
    company_id bigint NOT NULL,
    currency_code text DEFAULT 'USD'::text NOT NULL,
    category text DEFAULT 'other'::text NOT NULL,
    account_id bigint NOT NULL,
    analytic_distribution jsonb,
    move_line_id bigint,
    general_account_id bigint,
    source text DEFAULT 'manual'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_analytic_line
CREATE SEQUENCE public.account_analytic_line_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_analytic_line
ALTER SEQUENCE public.account_analytic_line_id_seq OWNED BY public.account_analytic_line.id;

-- ----------------------------------------------------------------------
-- TABLE: public.account_analytic_plan
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   description:               text                       
--   parent_id:                 bigint                     
--   parent_path:               text                       
--   root_id:                   bigint                     
--   sequence:                  integer                    default=10
--   color:                     integer                    
--   default_applicability:     text                       default='optional'
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.account_analytic_plan
CREATE TABLE public.account_analytic_plan (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    description text,
    parent_id bigint,
    parent_path text,
    root_id bigint,
    sequence integer DEFAULT 10,
    color integer,
    default_applicability text DEFAULT 'optional'::text NOT NULL,
    active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : account_analytic_plan
CREATE SEQUENCE public.account_analytic_plan_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : account_analytic_plan
ALTER SEQUENCE public.account_analytic_plan_id_seq OWNED BY public.account_analytic_plan.id;

-- ==============================================================================
-- SECTION: fleet      Fleet / Vehicles (الأسطول)
-- ------------------------------------------------------------------------------
-- Tables in this section: 12
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_service_types
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   category:                  varchar(16)                default='service'
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_service_types
CREATE TABLE public.fleet_service_types (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    category character varying(16) DEFAULT 'service'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_service_types
CREATE SEQUENCE public.fleet_service_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_service_types
ALTER SEQUENCE public.fleet_service_types_id_seq OWNED BY public.fleet_service_types.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_assignation_logs
--   id:                        bigint                     PK NOT NULL
--   vehicle_id:                bigint                     NOT NULL
--   driver_id:                 bigint                     NOT NULL
--   date_start:                date                       
--   date_end:                  date                       
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_assignation_logs
CREATE TABLE public.fleet_vehicle_assignation_logs (
    id bigint NOT NULL,
    vehicle_id bigint NOT NULL,
    driver_id bigint NOT NULL,
    date_start date,
    date_end date,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_assignation_logs
CREATE SEQUENCE public.fleet_vehicle_assignation_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_assignation_logs
ALTER SEQUENCE public.fleet_vehicle_assignation_logs_id_seq OWNED BY public.fleet_vehicle_assignation_logs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_brands
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   image_128:                 bytea                      
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_brands
CREATE TABLE public.fleet_vehicle_brands (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    image_128 bytea,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_brands
CREATE SEQUENCE public.fleet_vehicle_brands_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_brands
ALTER SEQUENCE public.fleet_vehicle_brands_id_seq OWNED BY public.fleet_vehicle_brands.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_log_contracts
--   id:                        bigint                     PK NOT NULL
--   vehicle_id:                bigint                     NOT NULL
--   start_date:                date                       NOT NULL
--   expiration_date:           date                       
--   cost_generated:            numeric(20,4)              default=0.0
--   cost_frequency:            varchar(20)                default='monthly'
--   ins_ref:                   varchar(64)                
--   insurer_id:                bigint                     
--   state:                     varchar(20)                default='open'
--   notes:                     text                       
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   user_id:                   bigint                     
--   date:                      date                       
--   name:                      varchar(255)               
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_log_contracts
CREATE TABLE public.fleet_vehicle_log_contracts (
    id bigint NOT NULL,
    vehicle_id bigint NOT NULL,
    start_date date NOT NULL,
    expiration_date date,
    cost_generated numeric(20,4) DEFAULT 0.0,
    cost_frequency character varying(20) DEFAULT 'monthly'::character varying,
    ins_ref character varying(64),
    insurer_id bigint,
    state character varying(20) DEFAULT 'open'::character varying,
    notes text,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    user_id bigint,
    date date,
    name character varying(255)
);

-- CREATE SEQUENCE : fleet_vehicle_log_contracts
CREATE SEQUENCE public.fleet_vehicle_log_contracts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_log_contracts
ALTER SEQUENCE public.fleet_vehicle_log_contracts_id_seq OWNED BY public.fleet_vehicle_log_contracts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_log_services
--   id:                        bigint                     PK NOT NULL
--   vehicle_id:                bigint                     NOT NULL
--   description:               varchar(255)               
--   date:                      date                       default=CURRENT_DATE
--   amount:                    numeric(20,4)              default=0.0
--   vendor_id:                 bigint                     
--   odometer:                  float8                     
--   notes:                     text                       
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   service_type_id:           bigint                     
--   inv_ref:                   varchar(64)                
--   state:                     varchar(16)                default='new'
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_log_services
CREATE TABLE public.fleet_vehicle_log_services (
    id bigint NOT NULL,
    vehicle_id bigint NOT NULL,
    description character varying(255),
    date date DEFAULT CURRENT_DATE NOT NULL,
    amount numeric(20,4) DEFAULT 0.0,
    vendor_id bigint,
    odometer double precision,
    notes text,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    service_type_id bigint,
    inv_ref character varying(64),
    state character varying(16) DEFAULT 'new'::character varying NOT NULL
);

-- CREATE SEQUENCE : fleet_vehicle_log_services
CREATE SEQUENCE public.fleet_vehicle_log_services_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_log_services
ALTER SEQUENCE public.fleet_vehicle_log_services_id_seq OWNED BY public.fleet_vehicle_log_services.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_model_categories
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_model_categories
CREATE TABLE public.fleet_vehicle_model_categories (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_model_categories
CREATE SEQUENCE public.fleet_vehicle_model_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_model_categories
ALTER SEQUENCE public.fleet_vehicle_model_categories_id_seq OWNED BY public.fleet_vehicle_model_categories.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_models
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   brand_id:                  bigint                     NOT NULL
--   category_id:               bigint                     
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_models
CREATE TABLE public.fleet_vehicle_models (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    brand_id bigint NOT NULL,
    category_id bigint,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_models
CREATE SEQUENCE public.fleet_vehicle_models_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_models
ALTER SEQUENCE public.fleet_vehicle_models_id_seq OWNED BY public.fleet_vehicle_models.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_odometers
--   id:                        bigint                     PK NOT NULL
--   vehicle_id:                bigint                     NOT NULL
--   date:                      date                       default=CURRENT_DATE
--   value:                     float8                     NOT NULL
--   unit:                      varchar(10)                default='kilometers'
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_odometers
CREATE TABLE public.fleet_vehicle_odometers (
    id bigint NOT NULL,
    vehicle_id bigint NOT NULL,
    date date DEFAULT CURRENT_DATE NOT NULL,
    value double precision NOT NULL,
    unit character varying(10) DEFAULT 'kilometers'::character varying,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_odometers
CREATE SEQUENCE public.fleet_vehicle_odometers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_odometers
ALTER SEQUENCE public.fleet_vehicle_odometers_id_seq OWNED BY public.fleet_vehicle_odometers.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_states
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   fold:                      boolean                    default=false
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_states
CREATE TABLE public.fleet_vehicle_states (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    fold boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_states
CREATE SEQUENCE public.fleet_vehicle_states_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_states
ALTER SEQUENCE public.fleet_vehicle_states_id_seq OWNED BY public.fleet_vehicle_states.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_tag_rel
--   vehicle_id:                bigint                     PK NOT NULL
--   tag_id:                    bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_tag_rel
CREATE TABLE public.fleet_vehicle_tag_rel (
    vehicle_id bigint NOT NULL,
    tag_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicle_tags
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   color:                     integer                    
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicle_tags
CREATE TABLE public.fleet_vehicle_tags (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    color integer,
    created_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : fleet_vehicle_tags
CREATE SEQUENCE public.fleet_vehicle_tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicle_tags
ALTER SEQUENCE public.fleet_vehicle_tags_id_seq OWNED BY public.fleet_vehicle_tags.id;

-- ----------------------------------------------------------------------
-- TABLE: public.fleet_vehicles
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               
--   license_plate:             varchar(32)                NOT NULL
--   model_id:                  bigint                     NOT NULL
--   driver_id:                 bigint                     
--   future_driver_id:          bigint                     
--   vin_sn:                    varchar(64)                
--   acquisition_date:          date                       
--   first_contract_date:       date                       
--   odometer:                  float8                     default=0.0
--   odometer_unit:             varchar(10)                default='kilometers'
--   fuel_type:                 varchar(20)                
--   horsepower:                integer                    
--   horsepower_tax:            float8                     
--   seats:                     integer                    
--   doors:                     integer                    
--   color:                     varchar(32)                
--   location:                  varchar(128)               
--   state:                     varchar(20)                default='active'
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   state_id:                  bigint                     
--   manager_id:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.fleet_vehicles
CREATE TABLE public.fleet_vehicles (
    id bigint NOT NULL,
    name character varying(255),
    license_plate character varying(32) NOT NULL,
    model_id bigint NOT NULL,
    driver_id bigint,
    future_driver_id bigint,
    vin_sn character varying(64),
    acquisition_date date,
    first_contract_date date,
    odometer double precision DEFAULT 0.0,
    odometer_unit character varying(10) DEFAULT 'kilometers'::character varying,
    fuel_type character varying(20),
    horsepower integer,
    horsepower_tax double precision,
    seats integer,
    doors integer,
    color character varying(32),
    location character varying(128),
    state character varying(20) DEFAULT 'active'::character varying,
    active boolean DEFAULT true,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    state_id bigint,
    manager_id bigint
);

-- CREATE SEQUENCE : fleet_vehicles
CREATE SEQUENCE public.fleet_vehicles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : fleet_vehicles
ALTER SEQUENCE public.fleet_vehicles_id_seq OWNED BY public.fleet_vehicles.id;

-- ==============================================================================
-- SECTION: livechat   Livechat (المحادثة المباشرة)
-- ------------------------------------------------------------------------------
-- Tables in this section: 4
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.livechat_channel_users
--   channel_id:                bigint                     PK NOT NULL
--   user_id:                   bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.livechat_channel_users
CREATE TABLE public.livechat_channel_users (
    channel_id bigint NOT NULL,
    user_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.livechat_channels
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   welcome_msg:               text                       default='مرحباً بك! كيف يمكننا مساعدتك اليوم؟'
--   button_text:               varchar(64)                default='تحدث معنا'
--   header_color:              varchar(16)                default='#1E3A8A'
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.livechat_channels
CREATE TABLE public.livechat_channels (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    welcome_msg text DEFAULT 'مرحباً بك! كيف يمكننا مساعدتك اليوم؟'::text NOT NULL,
    button_text character varying(64) DEFAULT 'تحدث معنا'::character varying NOT NULL,
    header_color character varying(16) DEFAULT '#1E3A8A'::character varying NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : livechat_channels
CREATE SEQUENCE public.livechat_channels_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : livechat_channels
ALTER SEQUENCE public.livechat_channels_id_seq OWNED BY public.livechat_channels.id;

-- ----------------------------------------------------------------------
-- TABLE: public.livechat_messages
--   id:                        bigint                     PK NOT NULL
--   session_id:                bigint                     NOT NULL
--   sender_type:               varchar(16)                NOT NULL
--   sender_id:                 bigint                     
--   body:                      text                       NOT NULL
--   file_url:                  text                       
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.livechat_messages
CREATE TABLE public.livechat_messages (
    id bigint NOT NULL,
    session_id bigint NOT NULL,
    sender_type character varying(16) NOT NULL,
    sender_id bigint,
    body text NOT NULL,
    file_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : livechat_messages
CREATE SEQUENCE public.livechat_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : livechat_messages
ALTER SEQUENCE public.livechat_messages_id_seq OWNED BY public.livechat_messages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.livechat_sessions
--   id:                        bigint                     PK NOT NULL
--   channel_id:                bigint                     NOT NULL
--   operator_id:               bigint                     
--   visitor_uuid:              varchar(64)                NOT NULL
--   visitor_name:              varchar(128)               default='زائر'
--   visitor_email:             varchar(128)               
--   partner_id:                bigint                     
--   status:                    varchar(32)                default='active'
--   rating_score:              integer                    
--   rating_comment:            text                       
--   converted_ticket_id:       bigint                     
--   created_at:                timestamptz                default=now()
--   closed_at:                 timestamptz                
--   CONSTRAINT:                livechat_sessions_rating_score_check CHECK (((rating_score >= 1) AND (rating_score <= 5))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.livechat_sessions
CREATE TABLE public.livechat_sessions (
    id bigint NOT NULL,
    channel_id bigint NOT NULL,
    operator_id bigint,
    visitor_uuid character varying(64) NOT NULL,
    visitor_name character varying(128) DEFAULT 'زائر'::character varying NOT NULL,
    visitor_email character varying(128),
    partner_id bigint,
    status character varying(32) DEFAULT 'active'::character varying NOT NULL,
    rating_score integer,
    rating_comment text,
    converted_ticket_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    closed_at timestamp with time zone,
    CONSTRAINT livechat_sessions_rating_score_check CHECK (((rating_score >= 1) AND (rating_score <= 5)))
);

-- CREATE SEQUENCE : livechat_sessions
CREATE SEQUENCE public.livechat_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : livechat_sessions
ALTER SEQUENCE public.livechat_sessions_id_seq OWNED BY public.livechat_sessions.id;

-- ==============================================================================
-- SECTION: pos        POS / Point of Sale (نقاط البيع)
-- ------------------------------------------------------------------------------
-- Tables in this section: 13
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.pos_cash_movements
--   id:                        bigint                     PK NOT NULL
--   session_id:                bigint                     NOT NULL
--   type:                      varchar(8)                 NOT NULL
--   amount:                    numeric(15,4)              NOT NULL
--   reason:                    text                       NOT NULL
--   user_id:                   bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   CONSTRAINT:                pos_cash_movements_amount_check CHECK ((amount > (0)::numeric)) 
--   CONSTRAINT:                pos_cash_movements_type_check CHECK (((type)::text = ANY ((ARRAY['in'::varchar, 'out'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_cash_movements
CREATE TABLE public.pos_cash_movements (
    id bigint NOT NULL,
    session_id bigint NOT NULL,
    type character varying(8) NOT NULL,
    amount numeric(15,4) NOT NULL,
    reason text NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pos_cash_movements_amount_check CHECK ((amount > (0)::numeric)),
    CONSTRAINT pos_cash_movements_type_check CHECK (((type)::text = ANY ((ARRAY['in'::character varying, 'out'::character varying])::text[])))
);

-- CREATE SEQUENCE : pos_cash_movements
CREATE SEQUENCE public.pos_cash_movements_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_cash_movements
ALTER SEQUENCE public.pos_cash_movements_id_seq OWNED BY public.pos_cash_movements.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_config_payment_method_rel
--   config_id:                 bigint                     PK NOT NULL
--   payment_method_id:         bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_config_payment_method_rel
CREATE TABLE public.pos_config_payment_method_rel (
    config_id bigint NOT NULL,
    payment_method_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.pos_configs
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   warehouse_id:              bigint                     NOT NULL
--   stock_location_id:         bigint                     NOT NULL
--   journal_id:                bigint                     NOT NULL
--   invoice_journal_id:        bigint                     
--   module_pos_restaurant:     boolean                    default=false
--   update_stock_at_closing:   boolean                    default=true
--   allow_discount:            boolean                    default=true
--   manual_discount_limit:     numeric(5,2)               default=100.00
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_configs
CREATE TABLE public.pos_configs (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    warehouse_id bigint NOT NULL,
    stock_location_id bigint NOT NULL,
    journal_id bigint NOT NULL,
    invoice_journal_id bigint,
    module_pos_restaurant boolean DEFAULT false NOT NULL,
    update_stock_at_closing boolean DEFAULT true NOT NULL,
    allow_discount boolean DEFAULT true NOT NULL,
    manual_discount_limit numeric(5,2) DEFAULT 100.00 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_configs
CREATE SEQUENCE public.pos_configs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_configs
ALTER SEQUENCE public.pos_configs_id_seq OWNED BY public.pos_configs.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_kitchen_ticket_lines
--   id:                        bigint                     PK NOT NULL
--   ticket_id:                 bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   product_name:              varchar(256)               NOT NULL
--   qty:                       numeric(15,4)              NOT NULL
--   notes:                     text                       
--   CONSTRAINT:                pos_kitchen_ticket_lines_qty_check CHECK ((qty > (0)::numeric)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_kitchen_ticket_lines
CREATE TABLE public.pos_kitchen_ticket_lines (
    id bigint NOT NULL,
    ticket_id bigint NOT NULL,
    product_id bigint NOT NULL,
    product_name character varying(256) NOT NULL,
    qty numeric(15,4) NOT NULL,
    notes text,
    CONSTRAINT pos_kitchen_ticket_lines_qty_check CHECK ((qty > (0)::numeric))
);

-- CREATE SEQUENCE : pos_kitchen_ticket_lines
CREATE SEQUENCE public.pos_kitchen_ticket_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_kitchen_ticket_lines
ALTER SEQUENCE public.pos_kitchen_ticket_lines_id_seq OWNED BY public.pos_kitchen_ticket_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_kitchen_tickets
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   table_id:                  bigint                     
--   status:                    varchar(32)                default='pending'
--   course:                    varchar(32)                default='main'
--   notes:                     text                       
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_kitchen_tickets
CREATE TABLE public.pos_kitchen_tickets (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    table_id bigint,
    status character varying(32) DEFAULT 'pending'::character varying NOT NULL,
    course character varying(32) DEFAULT 'main'::character varying NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_kitchen_tickets
CREATE SEQUENCE public.pos_kitchen_tickets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_kitchen_tickets
ALTER SEQUENCE public.pos_kitchen_tickets_id_seq OWNED BY public.pos_kitchen_tickets.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_order_lines
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   qty:                       numeric(15,4)              NOT NULL
--   price_unit:                numeric(15,4)              default=0
--   discount:                  numeric(5,2)               default=0
--   tax_rate:                  numeric(5,2)               default=0
--   price_subtotal:            numeric(15,4)              default=0
--   price_subtotal_incl:       numeric(15,4)              default=0
--   customer_note:             text                       
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_order_lines
CREATE TABLE public.pos_order_lines (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    product_id bigint NOT NULL,
    qty numeric(15,4) NOT NULL,
    price_unit numeric(15,4) DEFAULT 0 NOT NULL,
    discount numeric(5,2) DEFAULT 0 NOT NULL,
    tax_rate numeric(5,2) DEFAULT 0 NOT NULL,
    price_subtotal numeric(15,4) DEFAULT 0 NOT NULL,
    price_subtotal_incl numeric(15,4) DEFAULT 0 NOT NULL,
    customer_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_order_lines
CREATE SEQUENCE public.pos_order_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_order_lines
ALTER SEQUENCE public.pos_order_lines_id_seq OWNED BY public.pos_order_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_orders
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   client_uuid:               varchar(64)                NOT NULL
--   session_id:                bigint                     NOT NULL
--   partner_id:                bigint                     
--   user_id:                   bigint                     NOT NULL
--   table_id:                  bigint                     
--   customer_count:            integer                    default=0
--   state:                     varchar(32)                default='draft'
--   amount_untaxed:            numeric(15,4)              default=0
--   amount_tax:                numeric(15,4)              default=0
--   amount_total:              numeric(15,4)              default=0
--   amount_paid:               numeric(15,4)              default=0
--   amount_return:             numeric(15,4)              default=0
--   tip_amount:                numeric(15,4)              default=0
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_orders
CREATE TABLE public.pos_orders (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    client_uuid character varying(64) NOT NULL,
    session_id bigint NOT NULL,
    partner_id bigint,
    user_id bigint NOT NULL,
    table_id bigint,
    customer_count integer DEFAULT 0 NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    amount_untaxed numeric(15,4) DEFAULT 0 NOT NULL,
    amount_tax numeric(15,4) DEFAULT 0 NOT NULL,
    amount_total numeric(15,4) DEFAULT 0 NOT NULL,
    amount_paid numeric(15,4) DEFAULT 0 NOT NULL,
    amount_return numeric(15,4) DEFAULT 0 NOT NULL,
    tip_amount numeric(15,4) DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_orders
CREATE SEQUENCE public.pos_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_orders
ALTER SEQUENCE public.pos_orders_id_seq OWNED BY public.pos_orders.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_payment_methods
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   journal_id:                bigint                     NOT NULL
--   is_cash_count:             boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_payment_methods
CREATE TABLE public.pos_payment_methods (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    journal_id bigint NOT NULL,
    is_cash_count boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : pos_payment_methods
CREATE SEQUENCE public.pos_payment_methods_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_payment_methods
ALTER SEQUENCE public.pos_payment_methods_id_seq OWNED BY public.pos_payment_methods.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_payments
--   id:                        bigint                     PK NOT NULL
--   order_id:                  bigint                     NOT NULL
--   session_id:                bigint                     NOT NULL
--   payment_method_id:         bigint                     NOT NULL
--   amount:                    numeric(15,4)              NOT NULL
--   payment_date:              timestamptz                default=now()
--   transaction_id:            varchar(128)               
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_payments
CREATE TABLE public.pos_payments (
    id bigint NOT NULL,
    order_id bigint NOT NULL,
    session_id bigint NOT NULL,
    payment_method_id bigint NOT NULL,
    amount numeric(15,4) NOT NULL,
    payment_date timestamp with time zone DEFAULT now() NOT NULL,
    transaction_id character varying(128)
);

-- CREATE SEQUENCE : pos_payments
CREATE SEQUENCE public.pos_payments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_payments
ALTER SEQUENCE public.pos_payments_id_seq OWNED BY public.pos_payments.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_sessions
--   id:                        bigint                     PK NOT NULL
--   config_id:                 bigint                     NOT NULL
--   user_id:                   bigint                     NOT NULL
--   name:                      varchar(64)                NOT NULL
--   state:                     varchar(32)                default='opening_control'
--   start_at:                  timestamptz                default=now()
--   stop_at:                   timestamptz                
--   cash_register_balance_start: numeric(15,4)              default=0
--   cash_register_balance_end: numeric(15,4)              default=0
--   cash_register_balance_real: numeric(15,4)              default=0
--   cash_register_difference:  numeric(15,4)              default=0
--   total_orders_count:        integer                    default=0
--   total_payments_amount:     numeric(15,4)              default=0
--   stock_picking_id:          bigint                     
--   account_move_id:           bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_sessions
CREATE TABLE public.pos_sessions (
    id bigint NOT NULL,
    config_id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(64) NOT NULL,
    state character varying(32) DEFAULT 'opening_control'::character varying NOT NULL,
    start_at timestamp with time zone DEFAULT now() NOT NULL,
    stop_at timestamp with time zone,
    cash_register_balance_start numeric(15,4) DEFAULT 0 NOT NULL,
    cash_register_balance_end numeric(15,4) DEFAULT 0 NOT NULL,
    cash_register_balance_real numeric(15,4) DEFAULT 0 NOT NULL,
    cash_register_difference numeric(15,4) DEFAULT 0 NOT NULL,
    total_orders_count integer DEFAULT 0 NOT NULL,
    total_payments_amount numeric(15,4) DEFAULT 0 NOT NULL,
    stock_picking_id bigint,
    account_move_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_sessions
CREATE SEQUENCE public.pos_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_sessions
ALTER SEQUENCE public.pos_sessions_id_seq OWNED BY public.pos_sessions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.pos_sync_batches
--   id:                        bigint                     PK NOT NULL
--   session_id:                bigint                     NOT NULL
--   idempotency_key:           varchar(128)               NOT NULL
--   accepted_count:            integer                    default=0
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.pos_sync_batches
CREATE TABLE public.pos_sync_batches (
    id bigint NOT NULL,
    session_id bigint NOT NULL,
    idempotency_key character varying(128) NOT NULL,
    accepted_count integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : pos_sync_batches
CREATE SEQUENCE public.pos_sync_batches_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : pos_sync_batches
ALTER SEQUENCE public.pos_sync_batches_id_seq OWNED BY public.pos_sync_batches.id;

-- ----------------------------------------------------------------------
-- TABLE: public.restaurant_floors
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   pos_config_id:             bigint                     NOT NULL
--   sequence:                  integer                    default=10
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.restaurant_floors
CREATE TABLE public.restaurant_floors (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    pos_config_id bigint NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : restaurant_floors
CREATE SEQUENCE public.restaurant_floors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : restaurant_floors
ALTER SEQUENCE public.restaurant_floors_id_seq OWNED BY public.restaurant_floors.id;

-- ----------------------------------------------------------------------
-- TABLE: public.restaurant_tables
--   id:                        bigint                     PK NOT NULL
--   floor_id:                  bigint                     NOT NULL
--   name:                      varchar(32)                NOT NULL
--   seats:                     integer                    default=4
--   shape:                     varchar(16)                default='square'
--   position_x:                numeric(10,2)              default=0
--   position_y:                numeric(10,2)              default=0
--   width:                     numeric(10,2)              default=100
--   height:                    numeric(10,2)              default=100
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                restaurant_tables_seats_check CHECK ((seats > 0)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.restaurant_tables
CREATE TABLE public.restaurant_tables (
    id bigint NOT NULL,
    floor_id bigint NOT NULL,
    name character varying(32) NOT NULL,
    seats integer DEFAULT 4 NOT NULL,
    shape character varying(16) DEFAULT 'square'::character varying NOT NULL,
    position_x numeric(10,2) DEFAULT 0 NOT NULL,
    position_y numeric(10,2) DEFAULT 0 NOT NULL,
    width numeric(10,2) DEFAULT 100 NOT NULL,
    height numeric(10,2) DEFAULT 100 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT restaurant_tables_seats_check CHECK ((seats > 0))
);

-- CREATE SEQUENCE : restaurant_tables
CREATE SEQUENCE public.restaurant_tables_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : restaurant_tables
ALTER SEQUENCE public.restaurant_tables_id_seq OWNED BY public.restaurant_tables.id;

-- ==============================================================================
-- SECTION: project    Project / Tasks / Timesheets (المشاريع)
-- ------------------------------------------------------------------------------
-- Tables in this section: 12
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.project_milestones
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   project_id:                bigint                     NOT NULL
--   date_deadline:             date                       
--   is_reached:                boolean                    default=false
--   reached_date:              date                       
--   sequence:                  integer                    default=10
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_milestones
CREATE TABLE public.project_milestones (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    project_id bigint NOT NULL,
    date_deadline date,
    is_reached boolean DEFAULT false NOT NULL,
    reached_date date,
    sequence integer DEFAULT 10 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : project_milestones
CREATE SEQUENCE public.project_milestones_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_milestones
ALTER SEQUENCE public.project_milestones_id_seq OWNED BY public.project_milestones.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_project_stages
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   fold:                      boolean                    default=false
--   color:                     integer                    default=0
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_project_stages
CREATE TABLE public.project_project_stages (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    fold boolean DEFAULT false NOT NULL,
    color integer DEFAULT 0 NOT NULL,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : project_project_stages
CREATE SEQUENCE public.project_project_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_project_stages
ALTER SEQUENCE public.project_project_stages_id_seq OWNED BY public.project_project_stages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_projects
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   description:               text                       
--   partner_id:                bigint                     
--   manager_id:                bigint                     
--   stage_id:                  bigint                     
--   date_start:                date                       
--   date_end:                  date                       
--   active:                    boolean                    default=true
--   allow_milestones:          boolean                    default=false
--   allow_subtasks:            boolean                    default=true
--   allow_dependencies:        boolean                    default=false
--   analytic_account_id:       bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                chk_project_projects_dates CHECK (((date_end IS NULL) OR (date_start IS NULL) OR (date_end >= date_start))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_projects
CREATE TABLE public.project_projects (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    description text,
    partner_id bigint,
    manager_id bigint,
    stage_id bigint,
    date_start date,
    date_end date,
    active boolean DEFAULT true NOT NULL,
    allow_milestones boolean DEFAULT false NOT NULL,
    allow_subtasks boolean DEFAULT true NOT NULL,
    allow_dependencies boolean DEFAULT false NOT NULL,
    analytic_account_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT chk_project_projects_dates CHECK (((date_end IS NULL) OR (date_start IS NULL) OR (date_end >= date_start)))
);

-- CREATE SEQUENCE : project_projects
CREATE SEQUENCE public.project_projects_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_projects
ALTER SEQUENCE public.project_projects_id_seq OWNED BY public.project_projects.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_assignees
--   task_id:                   bigint                     PK NOT NULL
--   user_id:                   bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_assignees
CREATE TABLE public.project_task_assignees (
    task_id bigint NOT NULL,
    user_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_dependencies
--   task_id:                   bigint                     PK NOT NULL
--   depends_on_task_id:        bigint                     PK NOT NULL
--   CONSTRAINT:                chk_project_task_dependencies_not_self CHECK ((task_id <> depends_on_task_id)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_dependencies
CREATE TABLE public.project_task_dependencies (
    task_id bigint NOT NULL,
    depends_on_task_id bigint NOT NULL,
    CONSTRAINT chk_project_task_dependencies_not_self CHECK ((task_id <> depends_on_task_id))
);

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_tag_rel
--   task_id:                   bigint                     PK NOT NULL
--   tag_id:                    bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_tag_rel
CREATE TABLE public.project_task_tag_rel (
    task_id bigint NOT NULL,
    tag_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_tags
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   color:                     integer                    default=0
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_tags
CREATE TABLE public.project_task_tags (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    color integer DEFAULT 0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : project_task_tags
CREATE SEQUENCE public.project_task_tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_task_tags
ALTER SEQUENCE public.project_task_tags_id_seq OWNED BY public.project_task_tags.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_timers
--   id:                        bigint                     PK NOT NULL
--   task_id:                   bigint                     NOT NULL
--   employee_id:               bigint                     NOT NULL
--   start_time:                timestamptz                NOT NULL
--   is_running:                boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_timers
CREATE TABLE public.project_task_timers (
    id bigint NOT NULL,
    task_id bigint NOT NULL,
    employee_id bigint NOT NULL,
    start_time timestamp with time zone NOT NULL,
    is_running boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : project_task_timers
CREATE SEQUENCE public.project_task_timers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_task_timers
ALTER SEQUENCE public.project_task_timers_id_seq OWNED BY public.project_task_timers.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_type_projects
--   task_type_id:              bigint                     PK NOT NULL
--   project_id:                bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_type_projects
CREATE TABLE public.project_task_type_projects (
    task_type_id bigint NOT NULL,
    project_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.project_task_types
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=10
--   fold:                      boolean                    default=false
--   color:                     integer                    default=0
--   active:                    boolean                    default=true
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_task_types
CREATE TABLE public.project_task_types (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    fold boolean DEFAULT false NOT NULL,
    color integer DEFAULT 0 NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint
);

-- CREATE SEQUENCE : project_task_types
CREATE SEQUENCE public.project_task_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_task_types
ALTER SEQUENCE public.project_task_types_id_seq OWNED BY public.project_task_types.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_tasks
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   project_id:                bigint                     NOT NULL
--   stage_id:                  bigint                     NOT NULL
--   parent_id:                 bigint                     
--   priority:                  varchar(1)                 default='1'
--   date_deadline:             timestamptz                
--   date_assign:               timestamptz                
--   state:                     varchar(32)                default='in_progress'
--   description:               text                       
--   milestone_id:              bigint                     
--   sequence:                  integer                    default=10
--   allocated_hours:           float8                     default=0
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                chk_project_tasks_not_self_parent CHECK (((parent_id IS NULL) OR (parent_id <> id))) 
--   CONSTRAINT:                chk_project_tasks_priority CHECK (((priority)::text = ANY ((ARRAY['0'::varchar, '1'::varchar, '2'::varchar, '3'::varchar])::text[]))) 
--   CONSTRAINT:                chk_project_tasks_state CHECK (((state)::text = ANY ((ARRAY['in_progress'::varchar, 'changes_requested'::varchar, 'approved'::varchar, 'waiting'::varchar, 'done'::varchar, 'cancelled'::varchar])::text[]))) 
--   CONSTRAINT:                project_tasks_allocated_hours_check CHECK ((allocated_hours >= (0)::float8)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_tasks
CREATE TABLE public.project_tasks (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    project_id bigint NOT NULL,
    stage_id bigint NOT NULL,
    parent_id bigint,
    priority character varying(1) DEFAULT '1'::character varying NOT NULL,
    date_deadline timestamp with time zone,
    date_assign timestamp with time zone,
    state character varying(32) DEFAULT 'in_progress'::character varying NOT NULL,
    description text,
    milestone_id bigint,
    sequence integer DEFAULT 10 NOT NULL,
    allocated_hours double precision DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT chk_project_tasks_not_self_parent CHECK (((parent_id IS NULL) OR (parent_id <> id))),
    CONSTRAINT chk_project_tasks_priority CHECK (((priority)::text = ANY ((ARRAY['0'::character varying, '1'::character varying, '2'::character varying, '3'::character varying])::text[]))),
    CONSTRAINT chk_project_tasks_state CHECK (((state)::text = ANY ((ARRAY['in_progress'::character varying, 'changes_requested'::character varying, 'approved'::character varying, 'waiting'::character varying, 'done'::character varying, 'cancelled'::character varying])::text[]))),
    CONSTRAINT project_tasks_allocated_hours_check CHECK ((allocated_hours >= (0)::double precision))
);

-- CREATE SEQUENCE : project_tasks
CREATE SEQUENCE public.project_tasks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_tasks
ALTER SEQUENCE public.project_tasks_id_seq OWNED BY public.project_tasks.id;

-- ----------------------------------------------------------------------
-- TABLE: public.project_timesheets
--   id:                        bigint                     PK NOT NULL
--   project_id:                bigint                     NOT NULL
--   task_id:                   bigint                     
--   employee_id:               bigint                     NOT NULL
--   user_id:                   bigint                     NOT NULL
--   date:                      date                       NOT NULL
--   unit_amount:               numeric(6,2)               NOT NULL
--   name:                      text                       NOT NULL
--   hourly_cost:               numeric(15,4)              default=0
--   amount_total_cost:         numeric(15,4)              default=0
--   analytic_account_id:       bigint                     
--   state:                     varchar(32)                default='draft'
--   billable:                  boolean                    default=true
--   invoiced_timesheet:        boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                project_timesheets_unit_amount_check CHECK (((unit_amount > (0)::numeric) AND (unit_amount <= (24)::numeric))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.project_timesheets
CREATE TABLE public.project_timesheets (
    id bigint NOT NULL,
    project_id bigint NOT NULL,
    task_id bigint,
    employee_id bigint NOT NULL,
    user_id bigint NOT NULL,
    date date NOT NULL,
    unit_amount numeric(6,2) NOT NULL,
    name text NOT NULL,
    hourly_cost numeric(15,4) DEFAULT 0 NOT NULL,
    amount_total_cost numeric(15,4) DEFAULT 0 NOT NULL,
    analytic_account_id bigint,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    billable boolean DEFAULT true NOT NULL,
    invoiced_timesheet boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT project_timesheets_unit_amount_check CHECK (((unit_amount > (0)::numeric) AND (unit_amount <= (24)::numeric)))
);

-- CREATE SEQUENCE : project_timesheets
CREATE SEQUENCE public.project_timesheets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : project_timesheets
ALTER SEQUENCE public.project_timesheets_id_seq OWNED BY public.project_timesheets.id;

-- ==============================================================================
-- SECTION: other      Other / Core shared tables (جداول أخرى مشتركة)
-- ------------------------------------------------------------------------------
-- Tables in this section: 41
-- ==============================================================================

-- ----------------------------------------------------------------------
-- TABLE: public.helpdesk_sla_policies
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   team_id:                   bigint                     NOT NULL
--   priority:                  varchar(8)                 default='1'
--   max_hours_first_resp:      numeric(6,2)               default=4.0
--   max_hours_resolution:      numeric(6,2)               default=24.0
--   working_calendar_id:       bigint                     
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.helpdesk_sla_policies
CREATE TABLE public.helpdesk_sla_policies (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    team_id bigint NOT NULL,
    priority character varying(8) DEFAULT '1'::character varying NOT NULL,
    max_hours_first_resp numeric(6,2) DEFAULT 4.0 NOT NULL,
    max_hours_resolution numeric(6,2) DEFAULT 24.0 NOT NULL,
    working_calendar_id bigint,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : helpdesk_sla_policies
CREATE SEQUENCE public.helpdesk_sla_policies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : helpdesk_sla_policies
ALTER SEQUENCE public.helpdesk_sla_policies_id_seq OWNED BY public.helpdesk_sla_policies.id;

-- ----------------------------------------------------------------------
-- TABLE: public.helpdesk_stages
--   id:                        bigint                     PK NOT NULL
--   team_id:                   bigint                     NOT NULL
--   name:                      varchar(64)                NOT NULL
--   sequence:                  integer                    default=10
--   is_closed:                 boolean                    default=false
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.helpdesk_stages
CREATE TABLE public.helpdesk_stages (
    id bigint NOT NULL,
    team_id bigint NOT NULL,
    name character varying(64) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    is_closed boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : helpdesk_stages
CREATE SEQUENCE public.helpdesk_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : helpdesk_stages
ALTER SEQUENCE public.helpdesk_stages_id_seq OWNED BY public.helpdesk_stages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.helpdesk_teams
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   email:                     varchar(128)               
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.helpdesk_teams
CREATE TABLE public.helpdesk_teams (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    email character varying(128),
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : helpdesk_teams
CREATE SEQUENCE public.helpdesk_teams_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : helpdesk_teams
ALTER SEQUENCE public.helpdesk_teams_id_seq OWNED BY public.helpdesk_teams.id;

-- ----------------------------------------------------------------------
-- TABLE: public.helpdesk_tickets
--   id:                        bigint                     PK NOT NULL
--   number:                    varchar(64)                NOT NULL
--   name:                      varchar(256)               NOT NULL
--   description:               text                       NOT NULL
--   team_id:                   bigint                     NOT NULL
--   stage_id:                  bigint                     NOT NULL
--   priority:                  varchar(8)                 default='1'
--   partner_id:                bigint                     
--   partner_email:             varchar(128)               NOT NULL
--   partner_phone:             varchar(32)                
--   assigned_user_id:          bigint                     
--   sale_order_id:             bigint                     
--   stock_picking_id:          bigint                     
--   repair_order_id:           bigint                     
--   first_response_at:         timestamptz                
--   closed_at:                 timestamptz                
--   sla_breach:                boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.helpdesk_tickets
CREATE TABLE public.helpdesk_tickets (
    id bigint NOT NULL,
    number character varying(64) NOT NULL,
    name character varying(256) NOT NULL,
    description text NOT NULL,
    team_id bigint NOT NULL,
    stage_id bigint NOT NULL,
    priority character varying(8) DEFAULT '1'::character varying NOT NULL,
    partner_id bigint,
    partner_email character varying(128) NOT NULL,
    partner_phone character varying(32),
    assigned_user_id bigint,
    sale_order_id bigint,
    stock_picking_id bigint,
    repair_order_id bigint,
    first_response_at timestamp with time zone,
    closed_at timestamp with time zone,
    sla_breach boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : helpdesk_tickets
CREATE SEQUENCE public.helpdesk_tickets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : helpdesk_tickets
ALTER SEQUENCE public.helpdesk_tickets_id_seq OWNED BY public.helpdesk_tickets.id;

-- ----------------------------------------------------------------------
-- TABLE: public.ir_config_parameters
--   id:                        bigint                     PK NOT NULL
--   key:                       varchar(255)               NOT NULL
--   value:                     text                       default=''
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ir_config_parameters
CREATE TABLE public.ir_config_parameters (
    id bigint NOT NULL,
    key character varying(255) NOT NULL,
    value text DEFAULT ''::text NOT NULL,
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : ir_config_parameters
CREATE SEQUENCE public.ir_config_parameters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ir_config_parameters
ALTER SEQUENCE public.ir_config_parameters_id_seq OWNED BY public.ir_config_parameters.id;

-- ----------------------------------------------------------------------
-- TABLE: public.ir_sequences
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   code:                      varchar(100)               NOT NULL
--   prefix:                    varchar(50)                
--   suffix:                    varchar(50)                
--   padding:                   smallint                   default=5
--   increment_by:              integer                    default=1
--   start_number:              integer                    default=1
--   current_number:            integer                    default=0
--   sequence_type:             varchar(20)                default='normal'
--   date_range:                varchar(10)                
--   company_id:                bigint                     
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   CONSTRAINT:                ir_sequences_increment_by_check CHECK ((increment_by >= 1)) 
--   CONSTRAINT:                ir_sequences_padding_check CHECK (((padding >= 1) AND (padding <= 20))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ir_sequences
CREATE TABLE public.ir_sequences (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    code character varying(100) NOT NULL,
    prefix character varying(50),
    suffix character varying(50),
    padding smallint DEFAULT 5 NOT NULL,
    increment_by integer DEFAULT 1 NOT NULL,
    start_number integer DEFAULT 1 NOT NULL,
    current_number integer DEFAULT 0 NOT NULL,
    sequence_type character varying(20) DEFAULT 'normal'::character varying NOT NULL,
    date_range character varying(10),
    company_id bigint,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT ir_sequences_increment_by_check CHECK ((increment_by >= 1)),
    CONSTRAINT ir_sequences_padding_check CHECK (((padding >= 1) AND (padding <= 20)))
);

-- CREATE SEQUENCE : ir_sequences
CREATE SEQUENCE public.ir_sequences_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ir_sequences
ALTER SEQUENCE public.ir_sequences_id_seq OWNED BY public.ir_sequences.id;

-- ----------------------------------------------------------------------
-- TABLE: public.ir_translation
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   res_id:                    bigint                     
--   lang:                      varchar(10)                NOT NULL
--   type:                      varchar(20)                NOT NULL
--   src:                       text                       
--   value:                     text                       
--   module:                    varchar(100)               
--   state:                     varchar(20)                default='translated'
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.ir_translation
CREATE TABLE public.ir_translation (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    res_id bigint,
    lang character varying(10) NOT NULL,
    type character varying(20) NOT NULL,
    src text,
    value text,
    module character varying(100),
    state character varying(20) DEFAULT 'translated'::character varying,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : ir_translation
CREATE SEQUENCE public.ir_translation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : ir_translation
ALTER SEQUENCE public.ir_translation_id_seq OWNED BY public.ir_translation.id;

-- ----------------------------------------------------------------------
-- TABLE: public.knowledge_articles
--   id:                        bigint                     PK NOT NULL
--   category_id:               bigint                     NOT NULL
--   title:                     varchar(256)               NOT NULL
--   slug:                      varchar(256)               NOT NULL
--   content_html:              text                       NOT NULL
--   is_internal:               boolean                    default=false
--   view_count:                integer                    default=0
--   helpful_count:             integer                    default=0
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.knowledge_articles
CREATE TABLE public.knowledge_articles (
    id bigint NOT NULL,
    category_id bigint NOT NULL,
    title character varying(256) NOT NULL,
    slug character varying(256) NOT NULL,
    content_html text NOT NULL,
    is_internal boolean DEFAULT false NOT NULL,
    view_count integer DEFAULT 0 NOT NULL,
    helpful_count integer DEFAULT 0 NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : knowledge_articles
CREATE SEQUENCE public.knowledge_articles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : knowledge_articles
ALTER SEQUENCE public.knowledge_articles_id_seq OWNED BY public.knowledge_articles.id;

-- ----------------------------------------------------------------------
-- TABLE: public.knowledge_categories
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   sequence:                  integer                    default=10
--   company_id:                bigint                     NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.knowledge_categories
CREATE TABLE public.knowledge_categories (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL,
    company_id bigint NOT NULL
);

-- CREATE SEQUENCE : knowledge_categories
CREATE SEQUENCE public.knowledge_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : knowledge_categories
ALTER SEQUENCE public.knowledge_categories_id_seq OWNED BY public.knowledge_categories.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_activities
--   id:                        bigint                     PK NOT NULL
--   activity_type_id:          bigint                     NOT NULL
--   summary:                   varchar(255)               NOT NULL
--   note:                      text                       
--   date_deadline:             date                       NOT NULL
--   assigned_user_id:          bigint                     NOT NULL
--   res_model:                 varchar(128)               
--   res_id:                    bigint                     
--   active:                    boolean                    default=true
--   date_done:                 timestamptz                
--   feedback:                  text                       
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                chk_mail_activities_done_data CHECK ((active OR (date_done IS))) NOT NULL
--   CONSTRAINT:                chk_mail_activities_done_date CHECK (((date_done IS NULL) OR (date_done >= created_at))) 
--   CONSTRAINT:                chk_mail_activities_reference_pair CHECK (((res_model IS NULL) = (res_id IS NULL))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_activities
CREATE TABLE public.mail_activities (
    id bigint NOT NULL,
    activity_type_id bigint NOT NULL,
    summary character varying(255) NOT NULL,
    note text,
    date_deadline date NOT NULL,
    assigned_user_id bigint NOT NULL,
    res_model character varying(128),
    res_id bigint,
    active boolean DEFAULT true NOT NULL,
    date_done timestamp with time zone,
    feedback text,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT chk_mail_activities_done_data CHECK ((active OR (date_done IS NOT NULL))),
    CONSTRAINT chk_mail_activities_done_date CHECK (((date_done IS NULL) OR (date_done >= created_at))),
    CONSTRAINT chk_mail_activities_reference_pair CHECK (((res_model IS NULL) = (res_id IS NULL)))
);

-- CREATE SEQUENCE : mail_activities
CREATE SEQUENCE public.mail_activities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_activities
ALTER SEQUENCE public.mail_activities_id_seq OWNED BY public.mail_activities.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_activity_types
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   summary:                   varchar(255)               
--   res_model:                 varchar(128)               
--   category:                  varchar(32)                default='default'
--   delay_count:               integer                    default=0
--   delay_unit:                varchar(16)                default='days'
--   icon:                      varchar(128)               
--   sequence:                  integer                    default=10
--   default_note:              text                       
--   active:                    boolean                    default=true
--   system_type:               boolean                    default=false
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   created_by:                bigint                     
--   updated_by:                bigint                     
--   CONSTRAINT:                chk_mail_activity_types_delay_unit CHECK (((delay_unit)::text = ANY ((ARRAY['days'::varchar, 'weeks'::varchar, 'months'::varchar])::text[]))) 
--   CONSTRAINT:                mail_activity_types_delay_count_check CHECK ((delay_count >= 0)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_activity_types
CREATE TABLE public.mail_activity_types (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    summary character varying(255),
    res_model character varying(128),
    category character varying(32) DEFAULT 'default'::character varying NOT NULL,
    delay_count integer DEFAULT 0 NOT NULL,
    delay_unit character varying(16) DEFAULT 'days'::character varying NOT NULL,
    icon character varying(128),
    sequence integer DEFAULT 10 NOT NULL,
    default_note text,
    active boolean DEFAULT true NOT NULL,
    system_type boolean DEFAULT false NOT NULL,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by bigint,
    updated_by bigint,
    CONSTRAINT chk_mail_activity_types_delay_unit CHECK (((delay_unit)::text = ANY ((ARRAY['days'::character varying, 'weeks'::character varying, 'months'::character varying])::text[]))),
    CONSTRAINT mail_activity_types_delay_count_check CHECK ((delay_count >= 0))
);

-- CREATE SEQUENCE : mail_activity_types
CREATE SEQUENCE public.mail_activity_types_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_activity_types
ALTER SEQUENCE public.mail_activity_types_id_seq OWNED BY public.mail_activity_types.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_email_queue
--   id:                        bigint                     PK NOT NULL
--   notification_id:           bigint                     
--   recipient_email:           varchar(320)               NOT NULL
--   subject:                   varchar(255)               NOT NULL
--   body:                      text                       NOT NULL
--   status:                    varchar(16)                default='queued'
--   attempts:                  integer                    default=0
--   next_attempt_at:           timestamptz                default=now()
--   last_error:                text                       
--   sent_at:                   timestamptz                
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                chk_mail_email_queue_status CHECK (((status)::text = ANY ((ARRAY['queued'::varchar, 'processing'::varchar, 'sent'::varchar, 'failed'::varchar, 'dead_letter'::varchar])::text[]))) 
--   CONSTRAINT:                mail_email_queue_attempts_check CHECK ((attempts >= 0)) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_email_queue
CREATE TABLE public.mail_email_queue (
    id bigint NOT NULL,
    notification_id bigint,
    recipient_email character varying(320) NOT NULL,
    subject character varying(255) NOT NULL,
    body text NOT NULL,
    status character varying(16) DEFAULT 'queued'::character varying NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    next_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    last_error text,
    sent_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT chk_mail_email_queue_status CHECK (((status)::text = ANY ((ARRAY['queued'::character varying, 'processing'::character varying, 'sent'::character varying, 'failed'::character varying, 'dead_letter'::character varying])::text[]))),
    CONSTRAINT mail_email_queue_attempts_check CHECK ((attempts >= 0))
);

-- CREATE SEQUENCE : mail_email_queue
CREATE SEQUENCE public.mail_email_queue_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_email_queue
ALTER SEQUENCE public.mail_email_queue_id_seq OWNED BY public.mail_email_queue.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_followers
--   id:                        bigint                     PK NOT NULL
--   res_model:                 varchar(128)               NOT NULL
--   res_id:                    bigint                     NOT NULL
--   partner_id:                bigint                     
--   user_id:                   bigint                     
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                chk_mail_followers_target CHECK (((partner_id IS) OR (user_id IS))) NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_followers
CREATE TABLE public.mail_followers (
    id bigint NOT NULL,
    res_model character varying(128) NOT NULL,
    res_id bigint NOT NULL,
    partner_id bigint,
    user_id bigint,
    company_id bigint NOT NULL,
    CONSTRAINT chk_mail_followers_target CHECK (((partner_id IS NOT NULL) OR (user_id IS NOT NULL)))
);

-- CREATE SEQUENCE : mail_followers
CREATE SEQUENCE public.mail_followers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_followers
ALTER SEQUENCE public.mail_followers_id_seq OWNED BY public.mail_followers.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_followers_subtypes_rel
--   follower_id:               bigint                     PK NOT NULL
--   subtype_id:                bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_followers_subtypes_rel
CREATE TABLE public.mail_followers_subtypes_rel (
    follower_id bigint NOT NULL,
    subtype_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.mail_message_subtypes
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   res_model:                 varchar(128)               
--   description:               text                       
--   internal:                  boolean                    default=false
--   default_subtype:           boolean                    default=true
--   sequence:                  integer                    default=10
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_message_subtypes
CREATE TABLE public.mail_message_subtypes (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    res_model character varying(128),
    description text,
    internal boolean DEFAULT false NOT NULL,
    default_subtype boolean DEFAULT true NOT NULL,
    sequence integer DEFAULT 10 NOT NULL
);

-- CREATE SEQUENCE : mail_message_subtypes
CREATE SEQUENCE public.mail_message_subtypes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_message_subtypes
ALTER SEQUENCE public.mail_message_subtypes_id_seq OWNED BY public.mail_message_subtypes.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_messages
--   id:                        bigint                     PK NOT NULL
--   subject:                   varchar(255)               
--   body:                      text                       NOT NULL
--   message_type:              varchar(32)                default='notification'
--   res_model:                 varchar(128)               
--   res_id:                    bigint                     
--   author_id:                 bigint                     
--   activity_id:               bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   subtype_id:                bigint                     
--   parent_id:                 bigint                     
--   CONSTRAINT:                chk_mail_messages_reference_pair CHECK (((res_model IS NULL) = (res_id IS NULL))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_messages
CREATE TABLE public.mail_messages (
    id bigint NOT NULL,
    subject character varying(255),
    body text NOT NULL,
    message_type character varying(32) DEFAULT 'notification'::character varying NOT NULL,
    res_model character varying(128),
    res_id bigint,
    author_id bigint,
    activity_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    subtype_id bigint,
    parent_id bigint,
    CONSTRAINT chk_mail_messages_reference_pair CHECK (((res_model IS NULL) = (res_id IS NULL)))
);

-- CREATE SEQUENCE : mail_messages
CREATE SEQUENCE public.mail_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_messages
ALTER SEQUENCE public.mail_messages_id_seq OWNED BY public.mail_messages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_notifications
--   id:                        bigint                     PK NOT NULL
--   message_id:                bigint                     
--   activity_id:               bigint                     
--   recipient_user_id:         bigint                     NOT NULL
--   notification_type:         varchar(16)                default='inbox'
--   status:                    varchar(16)                default='unread'
--   read_at:                   timestamptz                
--   email:                     varchar(320)               
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   company_id:                bigint                     NOT NULL
--   CONSTRAINT:                chk_mail_notifications_read_at CHECK ((((status)::text = 'read'::text) = (read_at IS))) NOT NULL
--   CONSTRAINT:                chk_mail_notifications_status CHECK (((status)::text = ANY ((ARRAY['unread'::varchar, 'read'::varchar, 'queued'::varchar, 'sent'::varchar, 'failed'::varchar])::text[]))) 
--   CONSTRAINT:                chk_mail_notifications_type CHECK (((notification_type)::text = ANY ((ARRAY['inbox'::varchar, 'email'::varchar])::text[]))) 
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_notifications
CREATE TABLE public.mail_notifications (
    id bigint NOT NULL,
    message_id bigint,
    activity_id bigint,
    recipient_user_id bigint NOT NULL,
    notification_type character varying(16) DEFAULT 'inbox'::character varying NOT NULL,
    status character varying(16) DEFAULT 'unread'::character varying NOT NULL,
    read_at timestamp with time zone,
    email character varying(320),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    company_id bigint NOT NULL,
    CONSTRAINT chk_mail_notifications_read_at CHECK ((((status)::text = 'read'::text) = (read_at IS NOT NULL))),
    CONSTRAINT chk_mail_notifications_status CHECK (((status)::text = ANY ((ARRAY['unread'::character varying, 'read'::character varying, 'queued'::character varying, 'sent'::character varying, 'failed'::character varying])::text[]))),
    CONSTRAINT chk_mail_notifications_type CHECK (((notification_type)::text = ANY ((ARRAY['inbox'::character varying, 'email'::character varying])::text[])))
);

-- CREATE SEQUENCE : mail_notifications
CREATE SEQUENCE public.mail_notifications_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_notifications
ALTER SEQUENCE public.mail_notifications_id_seq OWNED BY public.mail_notifications.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mail_tracking_values
--   id:                        bigint                     PK NOT NULL
--   message_id:                bigint                     NOT NULL
--   field_name:                varchar(64)                NOT NULL
--   field_desc:                varchar(128)               NOT NULL
--   old_value_text:            text                       
--   new_value_text:            text                       
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mail_tracking_values
CREATE TABLE public.mail_tracking_values (
    id bigint NOT NULL,
    message_id bigint NOT NULL,
    field_name character varying(64) NOT NULL,
    field_desc character varying(128) NOT NULL,
    old_value_text text,
    new_value_text text
);

-- CREATE SEQUENCE : mail_tracking_values
CREATE SEQUENCE public.mail_tracking_values_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mail_tracking_values
ALTER SEQUENCE public.mail_tracking_values_id_seq OWNED BY public.mail_tracking_values.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mailing_contacts
--   id:                        bigint                     PK NOT NULL
--   partner_id:                bigint                     
--   email:                     varchar(128)               NOT NULL
--   mobile:                    varchar(32)                
--   name:                      varchar(128)               NOT NULL
--   is_opt_out:                boolean                    default=false
--   is_blacklist:              boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mailing_contacts
CREATE TABLE public.mailing_contacts (
    id bigint NOT NULL,
    partner_id bigint,
    email character varying(128) NOT NULL,
    mobile character varying(32),
    name character varying(128) NOT NULL,
    is_opt_out boolean DEFAULT false NOT NULL,
    is_blacklist boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : mailing_contacts
CREATE SEQUENCE public.mailing_contacts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mailing_contacts
ALTER SEQUENCE public.mailing_contacts_id_seq OWNED BY public.mailing_contacts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mailing_list_contact_rel
--   list_id:                   bigint                     PK NOT NULL
--   contact_id:                bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mailing_list_contact_rel
CREATE TABLE public.mailing_list_contact_rel (
    list_id bigint NOT NULL,
    contact_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.mailing_lists
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   is_public:                 boolean                    default=false
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mailing_lists
CREATE TABLE public.mailing_lists (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    is_public boolean DEFAULT false NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : mailing_lists
CREATE SEQUENCE public.mailing_lists_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mailing_lists
ALTER SEQUENCE public.mailing_lists_id_seq OWNED BY public.mailing_lists.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mailing_traces
--   id:                        bigint                     PK NOT NULL
--   mailing_id:                bigint                     NOT NULL
--   contact_id:                bigint                     NOT NULL
--   email:                     varchar(128)               NOT NULL
--   sent_at:                   timestamptz                default=now()
--   delivered_at:              timestamptz                
--   opened_at:                 timestamptz                
--   clicked_at:                timestamptz                
--   bounced_at:                timestamptz                
--   bounce_reason:             text                       
--   tracking_code:             varchar(64)                NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mailing_traces
CREATE TABLE public.mailing_traces (
    id bigint NOT NULL,
    mailing_id bigint NOT NULL,
    contact_id bigint NOT NULL,
    email character varying(128) NOT NULL,
    sent_at timestamp with time zone DEFAULT now() NOT NULL,
    delivered_at timestamp with time zone,
    opened_at timestamp with time zone,
    clicked_at timestamp with time zone,
    bounced_at timestamp with time zone,
    bounce_reason text,
    tracking_code character varying(64) NOT NULL
);

-- CREATE SEQUENCE : mailing_traces
CREATE SEQUENCE public.mailing_traces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mailing_traces
ALTER SEQUENCE public.mailing_traces_id_seq OWNED BY public.mailing_traces.id;

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_equipment
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   category_id:               bigint                     
--   technician_user_id:        bigint                     
--   owner_user_id:             bigint                     
--   employee_id:               bigint                     
--   department_id:             bigint                     
--   assign_to:                 varchar(20)                default='employee'
--   location_id:               bigint                     
--   serial_no:                 varchar(255)               
--   model:                     varchar(255)               
--   warranty_date:             date                       
--   effective_date:            date                       
--   next_action_date:          date                       
--   period:                    integer                    default=0
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   team_id:                   bigint                     
--   partner_id:                bigint                     
--   partner_ref:               varchar(64)                
--   cost:                      numeric(20,4)              default=0.0
--   notes:                     text                       
--   assign_date:               date                       
--   scrap_date:                date                       
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_equipment
CREATE TABLE public.maintenance_equipment (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    category_id bigint,
    technician_user_id bigint,
    owner_user_id bigint,
    employee_id bigint,
    department_id bigint,
    assign_to character varying(20) DEFAULT 'employee'::character varying,
    location_id bigint,
    serial_no character varying(255),
    model character varying(255),
    warranty_date date,
    effective_date date,
    next_action_date date,
    period integer DEFAULT 0,
    active boolean DEFAULT true,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    team_id bigint,
    partner_id bigint,
    partner_ref character varying(64),
    cost numeric(20,4) DEFAULT 0.0,
    notes text,
    assign_date date,
    scrap_date date
);

-- CREATE SEQUENCE : maintenance_equipment
CREATE SEQUENCE public.maintenance_equipment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : maintenance_equipment
ALTER SEQUENCE public.maintenance_equipment_id_seq OWNED BY public.maintenance_equipment.id;

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_equipment_categories
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   color:                     integer                    
--   active:                    boolean                    default=true
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_equipment_categories
CREATE TABLE public.maintenance_equipment_categories (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    color integer,
    active boolean DEFAULT true,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : maintenance_equipment_categories
CREATE SEQUENCE public.maintenance_equipment_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : maintenance_equipment_categories
ALTER SEQUENCE public.maintenance_equipment_categories_id_seq OWNED BY public.maintenance_equipment_categories.id;

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_requests
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   equipment_id:              bigint                     
--   request_date:              date                       default=CURRENT_DATE
--   close_date:                date                       
--   schedule_date:             timestamptz                
--   maintenance_type:          varchar(20)                default='corrective'
--   priority:                  varchar(1)                 default='0'
--   stage_id:                  bigint                     
--   technician_user_id:        bigint                     
--   owner_user_id:             bigint                     
--   employee_id:               bigint                     
--   department_id:             bigint                     
--   duration:                  float8                     default=0.0
--   description:               text                       
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
--   team_id:                   bigint                     
--   kanban_state:              varchar(20)                default='normal'
--   schedule_end:              timestamptz                
--   recurring_maintenance:     boolean                    default=false
--   repeat_interval:           integer                    default=1
--   repeat_unit:               varchar(16)                default='week'
--   repeat_type:               varchar(16)                default='forever'
--   repeat_until:              date                       
--   archived:                  boolean                    default=false
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_requests
CREATE TABLE public.maintenance_requests (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    equipment_id bigint,
    request_date date DEFAULT CURRENT_DATE NOT NULL,
    close_date date,
    schedule_date timestamp with time zone,
    maintenance_type character varying(20) DEFAULT 'corrective'::character varying,
    priority character varying(1) DEFAULT '0'::character varying,
    stage_id bigint,
    technician_user_id bigint,
    owner_user_id bigint,
    employee_id bigint,
    department_id bigint,
    duration double precision DEFAULT 0.0,
    description text,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    team_id bigint,
    kanban_state character varying(20) DEFAULT 'normal'::character varying NOT NULL,
    schedule_end timestamp with time zone,
    recurring_maintenance boolean DEFAULT false NOT NULL,
    repeat_interval integer DEFAULT 1 NOT NULL,
    repeat_unit character varying(16) DEFAULT 'week'::character varying NOT NULL,
    repeat_type character varying(16) DEFAULT 'forever'::character varying NOT NULL,
    repeat_until date,
    archived boolean DEFAULT false NOT NULL
);

-- CREATE SEQUENCE : maintenance_requests
CREATE SEQUENCE public.maintenance_requests_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : maintenance_requests
ALTER SEQUENCE public.maintenance_requests_id_seq OWNED BY public.maintenance_requests.id;

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_stages
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   sequence:                  integer                    default=0
--   fold:                      boolean                    default=false
--   done:                      boolean                    default=false
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_stages
CREATE TABLE public.maintenance_stages (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    sequence integer DEFAULT 0,
    fold boolean DEFAULT false,
    done boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : maintenance_stages
CREATE SEQUENCE public.maintenance_stages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : maintenance_stages
ALTER SEQUENCE public.maintenance_stages_id_seq OWNED BY public.maintenance_stages.id;

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_team_members
--   team_id:                   bigint                     PK NOT NULL
--   user_id:                   bigint                     PK NOT NULL
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_team_members
CREATE TABLE public.maintenance_team_members (
    team_id bigint NOT NULL,
    user_id bigint NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.maintenance_teams
--   id:                        bigint                     PK NOT NULL
--   name:                      jsonb                      NOT NULL
--   color:                     integer                    
--   active:                    boolean                    default=true
--   company_id:                bigint                     
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.maintenance_teams
CREATE TABLE public.maintenance_teams (
    id bigint NOT NULL,
    name jsonb NOT NULL,
    color integer,
    active boolean DEFAULT true,
    company_id bigint,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- CREATE SEQUENCE : maintenance_teams
CREATE SEQUENCE public.maintenance_teams_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : maintenance_teams
ALTER SEQUENCE public.maintenance_teams_id_seq OWNED BY public.maintenance_teams.id;

-- ----------------------------------------------------------------------
-- TABLE: public.marketing_automation_activities
--   id:                        bigint                     PK NOT NULL
--   automation_id:             bigint                     NOT NULL
--   parent_id:                 bigint                     
--   action_type:               varchar(32)                NOT NULL
--   delay_hours:               integer                    default=0
--   condition_type:            varchar(32)                default='none'
--   template_id:               bigint                     
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.marketing_automation_activities
CREATE TABLE public.marketing_automation_activities (
    id bigint NOT NULL,
    automation_id bigint NOT NULL,
    parent_id bigint,
    action_type character varying(32) NOT NULL,
    delay_hours integer DEFAULT 0 NOT NULL,
    condition_type character varying(32) DEFAULT 'none'::character varying NOT NULL,
    template_id bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : marketing_automation_activities
CREATE SEQUENCE public.marketing_automation_activities_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : marketing_automation_activities
ALTER SEQUENCE public.marketing_automation_activities_id_seq OWNED BY public.marketing_automation_activities.id;

-- ----------------------------------------------------------------------
-- TABLE: public.marketing_automations
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   trigger_type:              varchar(64)                NOT NULL
--   target_model:              varchar(64)                NOT NULL
--   filter_json:               jsonb                      default='{}'
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.marketing_automations
CREATE TABLE public.marketing_automations (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    trigger_type character varying(64) NOT NULL,
    target_model character varying(64) NOT NULL,
    filter_json jsonb DEFAULT '{}'::jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : marketing_automations
CREATE SEQUENCE public.marketing_automations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : marketing_automations
ALTER SEQUENCE public.marketing_automations_id_seq OWNED BY public.marketing_automations.id;

-- ----------------------------------------------------------------------
-- TABLE: public.marketing_campaigns
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   user_id:                   bigint                     NOT NULL
--   utm_source:                varchar(64)                default='marketing'
--   utm_medium:                varchar(64)                default='email'
--   utm_campaign:              varchar(128)               NOT NULL
--   total_sent:                integer                    default=0
--   total_delivered:           integer                    default=0
--   total_opened:              integer                    default=0
--   total_clicked:             integer                    default=0
--   total_bounced:             integer                    default=0
--   total_revenue:             numeric(15,4)              default=0
--   state:                     varchar(32)                default='draft'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.marketing_campaigns
CREATE TABLE public.marketing_campaigns (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    user_id bigint NOT NULL,
    utm_source character varying(64) DEFAULT 'marketing'::character varying NOT NULL,
    utm_medium character varying(64) DEFAULT 'email'::character varying NOT NULL,
    utm_campaign character varying(128) NOT NULL,
    total_sent integer DEFAULT 0 NOT NULL,
    total_delivered integer DEFAULT 0 NOT NULL,
    total_opened integer DEFAULT 0 NOT NULL,
    total_clicked integer DEFAULT 0 NOT NULL,
    total_bounced integer DEFAULT 0 NOT NULL,
    total_revenue numeric(15,4) DEFAULT 0 NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : marketing_campaigns
CREATE SEQUENCE public.marketing_campaigns_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : marketing_campaigns
ALTER SEQUENCE public.marketing_campaigns_id_seq OWNED BY public.marketing_campaigns.id;

-- ----------------------------------------------------------------------
-- TABLE: public.mass_mailings
--   id:                        bigint                     PK NOT NULL
--   campaign_id:               bigint                     
--   subject:                   varchar(256)               NOT NULL
--   sender_name:               varchar(128)               NOT NULL
--   sender_email:              varchar(128)               NOT NULL
--   reply_to:                  varchar(128)               
--   body_html:                 text                       NOT NULL
--   scheduled_date:            timestamptz                
--   sent_date:                 timestamptz                
--   state:                     varchar(32)                default='draft'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.mass_mailings
CREATE TABLE public.mass_mailings (
    id bigint NOT NULL,
    campaign_id bigint,
    subject character varying(256) NOT NULL,
    sender_name character varying(128) NOT NULL,
    sender_email character varying(128) NOT NULL,
    reply_to character varying(128),
    body_html text NOT NULL,
    scheduled_date timestamp with time zone,
    sent_date timestamp with time zone,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : mass_mailings
CREATE SEQUENCE public.mass_mailings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : mass_mailings
ALTER SEQUENCE public.mass_mailings_id_seq OWNED BY public.mass_mailings.id;

-- ----------------------------------------------------------------------
-- TABLE: public.quality_alerts
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   product_id:                bigint                     NOT NULL
--   lot_id:                    bigint                     
--   picking_id:                bigint                     
--   description:               text                       NOT NULL
--   action_taken:              text                       
--   stage:                     varchar(32)                default='new'
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.quality_alerts
CREATE TABLE public.quality_alerts (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    product_id bigint NOT NULL,
    lot_id bigint,
    picking_id bigint,
    description text NOT NULL,
    action_taken text,
    stage character varying(32) DEFAULT 'new'::character varying NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : quality_alerts
CREATE SEQUENCE public.quality_alerts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : quality_alerts
ALTER SEQUENCE public.quality_alerts_id_seq OWNED BY public.quality_alerts.id;

-- ----------------------------------------------------------------------
-- TABLE: public.quality_control_points
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(128)               NOT NULL
--   product_id:                bigint                     
--   category_id:               bigint                     
--   trigger:                   varchar(32)                NOT NULL
--   test_type:                 varchar(32)                default='pass_fail'
--   norm_min:                  numeric(10,4)              
--   norm_max:                  numeric(10,4)              
--   instructions:              text                       
--   company_id:                bigint                     NOT NULL
--   active:                    boolean                    default=true
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.quality_control_points
CREATE TABLE public.quality_control_points (
    id bigint NOT NULL,
    name character varying(128) NOT NULL,
    product_id bigint,
    category_id bigint,
    trigger character varying(32) NOT NULL,
    test_type character varying(32) DEFAULT 'pass_fail'::character varying NOT NULL,
    norm_min numeric(10,4),
    norm_max numeric(10,4),
    instructions text,
    company_id bigint NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- CREATE SEQUENCE : quality_control_points
CREATE SEQUENCE public.quality_control_points_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : quality_control_points
ALTER SEQUENCE public.quality_control_points_id_seq OWNED BY public.quality_control_points.id;

-- ----------------------------------------------------------------------
-- TABLE: public.repair_order_lines
--   id:                        bigint                     PK NOT NULL
--   repair_id:                 bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   quantity:                  numeric(15,4)              default=1
--   price_unit:                numeric(15,4)              default=0
--   price_total:               numeric(15,4)              default=0
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.repair_order_lines
CREATE TABLE public.repair_order_lines (
    id bigint NOT NULL,
    repair_id bigint NOT NULL,
    product_id bigint NOT NULL,
    quantity numeric(15,4) DEFAULT 1 NOT NULL,
    price_unit numeric(15,4) DEFAULT 0 NOT NULL,
    price_total numeric(15,4) DEFAULT 0 NOT NULL
);

-- CREATE SEQUENCE : repair_order_lines
CREATE SEQUENCE public.repair_order_lines_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : repair_order_lines
ALTER SEQUENCE public.repair_order_lines_id_seq OWNED BY public.repair_order_lines.id;

-- ----------------------------------------------------------------------
-- TABLE: public.repair_orders
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(64)                NOT NULL
--   partner_id:                bigint                     NOT NULL
--   product_id:                bigint                     NOT NULL
--   product_lot_id:            bigint                     
--   warranty_check:            boolean                    default=false
--   state:                     varchar(32)                default='draft'
--   location_id:               bigint                     NOT NULL
--   location_dest_id:          bigint                     NOT NULL
--   amount_total:              numeric(15,4)              default=0
--   account_move_id:           bigint                     
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.repair_orders
CREATE TABLE public.repair_orders (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    partner_id bigint NOT NULL,
    product_id bigint NOT NULL,
    product_lot_id bigint,
    warranty_check boolean DEFAULT false NOT NULL,
    state character varying(32) DEFAULT 'draft'::character varying NOT NULL,
    location_id bigint NOT NULL,
    location_dest_id bigint NOT NULL,
    amount_total numeric(15,4) DEFAULT 0 NOT NULL,
    account_move_id bigint,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : repair_orders
CREATE SEQUENCE public.repair_orders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : repair_orders
ALTER SEQUENCE public.repair_orders_id_seq OWNED BY public.repair_orders.id;

-- ----------------------------------------------------------------------
-- TABLE: public.res_lang
--   id:                        bigint                     PK NOT NULL
--   name:                      varchar(100)               NOT NULL
--   code:                      varchar(10)                NOT NULL
--   iso_code:                  varchar(5)                 
--   direction:                 varchar(3)                 default='ltr'
--   date_format:               varchar(50)                default='%Y-%m-%d'
--   time_format:               varchar(50)                default='%H:%M:%S'
--   week_start:                integer                    default=1
--   decimal_point:             varchar(5)                 default='.'
--   thousands_sep:             varchar(5)                 default=','
--   active:                    boolean                    default=true
--   created_at:                timestamptz                default=now()
--   updated_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.res_lang
CREATE TABLE public.res_lang (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    code character varying(10) NOT NULL,
    iso_code character varying(5),
    direction character varying(3) DEFAULT 'ltr'::character varying NOT NULL,
    date_format character varying(50) DEFAULT '%Y-%m-%d'::character varying NOT NULL,
    time_format character varying(50) DEFAULT '%H:%M:%S'::character varying NOT NULL,
    week_start integer DEFAULT 1 NOT NULL,
    decimal_point character varying(5) DEFAULT '.'::character varying NOT NULL,
    thousands_sep character varying(5) DEFAULT ','::character varying NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : res_lang
CREATE SEQUENCE public.res_lang_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : res_lang
ALTER SEQUENCE public.res_lang_id_seq OWNED BY public.res_lang.id;

-- ----------------------------------------------------------------------
-- TABLE: public.schema_migrations
--   version:                   bigint                     PK NOT NULL
--   name:                      varchar(255)               NOT NULL
--   applied_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.schema_migrations
CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    name character varying(255) NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);

-- ----------------------------------------------------------------------
-- TABLE: public.survey_questions
--   id:                        bigint                     PK NOT NULL
--   survey_id:                 bigint                     NOT NULL
--   title:                     text                       NOT NULL
--   type:                      varchar(32)                NOT NULL
--   sequence:                  integer                    default=10
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.survey_questions
CREATE TABLE public.survey_questions (
    id bigint NOT NULL,
    survey_id bigint NOT NULL,
    title text NOT NULL,
    type character varying(32) NOT NULL,
    sequence integer DEFAULT 10 NOT NULL
);

-- CREATE SEQUENCE : survey_questions
CREATE SEQUENCE public.survey_questions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : survey_questions
ALTER SEQUENCE public.survey_questions_id_seq OWNED BY public.survey_questions.id;

-- ----------------------------------------------------------------------
-- TABLE: public.survey_surveys
--   id:                        bigint                     PK NOT NULL
--   title:                     varchar(256)               NOT NULL
--   description:               text                       
--   is_scoring:                boolean                    default=false
--   passing_score:             numeric(5,2)               default=70.0
--   active:                    boolean                    default=true
--   company_id:                bigint                     NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.survey_surveys
CREATE TABLE public.survey_surveys (
    id bigint NOT NULL,
    title character varying(256) NOT NULL,
    description text,
    is_scoring boolean DEFAULT false NOT NULL,
    passing_score numeric(5,2) DEFAULT 70.0,
    active boolean DEFAULT true NOT NULL,
    company_id bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- CREATE SEQUENCE : survey_surveys
CREATE SEQUENCE public.survey_surveys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- ALTER SEQUENCE ... OWNED BY : survey_surveys
ALTER SEQUENCE public.survey_surveys_id_seq OWNED BY public.survey_surveys.id;

-- ----------------------------------------------------------------------
-- TABLE: public.transactions
--   id:                        varchar(64)                PK NOT NULL
--   amount:                    numeric(15,2)              NOT NULL
--   type:                      varchar(16)                NOT NULL
--   description:               text                       NOT NULL
--   created_at:                timestamptz                default=now()
-- ----------------------------------------------------------------------

-- CREATE TABLE : public.transactions
CREATE TABLE public.transactions (
    id character varying(64) NOT NULL,
    amount numeric(15,2) NOT NULL,
    type character varying(16) NOT NULL,
    description text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

-- column default
ALTER TABLE ONLY public.account_accounts ALTER COLUMN id SET DEFAULT nextval('public.account_accounts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_bank_statement_lines ALTER COLUMN id SET DEFAULT nextval('public.account_bank_statement_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_bank_statements ALTER COLUMN id SET DEFAULT nextval('public.account_bank_statements_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_cash_roundings ALTER COLUMN id SET DEFAULT nextval('public.account_cash_roundings_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_full_reconciles ALTER COLUMN id SET DEFAULT nextval('public.account_full_reconciles_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_journals ALTER COLUMN id SET DEFAULT nextval('public.account_journals_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_move_lines ALTER COLUMN id SET DEFAULT nextval('public.account_move_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_moves ALTER COLUMN id SET DEFAULT nextval('public.account_moves_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_partial_reconciles ALTER COLUMN id SET DEFAULT nextval('public.account_partial_reconciles_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_payment_reconciliations ALTER COLUMN id SET DEFAULT nextval('public.account_payment_reconciliations_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_payment_term_lines ALTER COLUMN id SET DEFAULT nextval('public.account_payment_term_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_payment_terms ALTER COLUMN id SET DEFAULT nextval('public.account_payment_terms_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_payments ALTER COLUMN id SET DEFAULT nextval('public.account_payments_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_reconcile_model_lines ALTER COLUMN id SET DEFAULT nextval('public.account_reconcile_model_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_reconcile_models ALTER COLUMN id SET DEFAULT nextval('public.account_reconcile_models_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_taxes ALTER COLUMN id SET DEFAULT nextval('public.account_taxes_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.accounting_periods ALTER COLUMN id SET DEFAULT nextval('public.accounting_periods_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_provider_configs ALTER COLUMN id SET DEFAULT nextval('public.payment_provider_configs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_providers ALTER COLUMN id SET DEFAULT nextval('public.payment_providers_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_refunds ALTER COLUMN id SET DEFAULT nextval('public.payment_refunds_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_tokens ALTER COLUMN id SET DEFAULT nextval('public.payment_tokens_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_transactions ALTER COLUMN id SET DEFAULT nextval('public.payment_transactions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.payment_webhook_logs ALTER COLUMN id SET DEFAULT nextval('public.payment_webhook_logs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_currencies ALTER COLUMN id SET DEFAULT nextval('public.res_currencies_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_currency_rates ALTER COLUMN id SET DEFAULT nextval('public.res_currency_rates_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.delivery_carrier ALTER COLUMN id SET DEFAULT nextval('public.delivery_carrier_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.delivery_price_rule ALTER COLUMN id SET DEFAULT nextval('public.delivery_price_rule_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.delivery_zip_prefix ALTER COLUMN id SET DEFAULT nextval('public.delivery_zip_prefix_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_account_config ALTER COLUMN id SET DEFAULT nextval('public.stock_account_config_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_barcode_nomenclatures ALTER COLUMN id SET DEFAULT nextval('public.stock_barcode_nomenclatures_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_barcode_rules ALTER COLUMN id SET DEFAULT nextval('public.stock_barcode_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_landed_cost_lines ALTER COLUMN id SET DEFAULT nextval('public.stock_landed_cost_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_landed_costs ALTER COLUMN id SET DEFAULT nextval('public.stock_landed_costs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_locations ALTER COLUMN id SET DEFAULT nextval('public.stock_locations_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_lots ALTER COLUMN id SET DEFAULT nextval('public.stock_lots_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_move_lines ALTER COLUMN id SET DEFAULT nextval('public.stock_move_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_moves ALTER COLUMN id SET DEFAULT nextval('public.stock_moves_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_orderpoints ALTER COLUMN id SET DEFAULT nextval('public.stock_orderpoints_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_package_types ALTER COLUMN id SET DEFAULT nextval('public.stock_package_types_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_packages ALTER COLUMN id SET DEFAULT nextval('public.stock_packages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_pickings ALTER COLUMN id SET DEFAULT nextval('public.stock_pickings_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_procurement_groups ALTER COLUMN id SET DEFAULT nextval('public.stock_procurement_groups_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_putaway_rules ALTER COLUMN id SET DEFAULT nextval('public.stock_putaway_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_quants ALTER COLUMN id SET DEFAULT nextval('public.stock_quants_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_routes ALTER COLUMN id SET DEFAULT nextval('public.stock_routes_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_rules ALTER COLUMN id SET DEFAULT nextval('public.stock_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_storage_categories ALTER COLUMN id SET DEFAULT nextval('public.stock_storage_categories_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_storage_category_capacities ALTER COLUMN id SET DEFAULT nextval('public.stock_storage_category_capacities_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_valuation_adjustment_lines ALTER COLUMN id SET DEFAULT nextval('public.stock_valuation_adjustment_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.stock_warehouses ALTER COLUMN id SET DEFAULT nextval('public.stock_warehouses_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_attribute_values ALTER COLUMN id SET DEFAULT nextval('public.product_attribute_values_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_attributes ALTER COLUMN id SET DEFAULT nextval('public.product_attributes_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_categories ALTER COLUMN id SET DEFAULT nextval('public.product_categories_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_packagings ALTER COLUMN id SET DEFAULT nextval('public.product_packagings_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_pricelist_items ALTER COLUMN id SET DEFAULT nextval('public.product_pricelist_items_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_pricelists ALTER COLUMN id SET DEFAULT nextval('public.product_pricelists_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_templates ALTER COLUMN id SET DEFAULT nextval('public.product_templates_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_values ALTER COLUMN id SET DEFAULT nextval('public.product_values_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.product_variants ALTER COLUMN id SET DEFAULT nextval('public.product_variants_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.uom_uoms ALTER COLUMN id SET DEFAULT nextval('public.uom_uoms_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ecommerce_cart_lines ALTER COLUMN id SET DEFAULT nextval('public.ecommerce_cart_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ecommerce_carts ALTER COLUMN id SET DEFAULT nextval('public.ecommerce_carts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_card_history ALTER COLUMN id SET DEFAULT nextval('public.loyalty_card_history_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_cards ALTER COLUMN id SET DEFAULT nextval('public.loyalty_cards_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_mails ALTER COLUMN id SET DEFAULT nextval('public.loyalty_mails_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_programs ALTER COLUMN id SET DEFAULT nextval('public.loyalty_programs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_rewards ALTER COLUMN id SET DEFAULT nextval('public.loyalty_rewards_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.loyalty_rules ALTER COLUMN id SET DEFAULT nextval('public.loyalty_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.sale_order_coupon_points ALTER COLUMN id SET DEFAULT nextval('public.sale_order_coupon_points_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.sale_order_lines ALTER COLUMN id SET DEFAULT nextval('public.sale_order_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.sale_orders ALTER COLUMN id SET DEFAULT nextval('public.sale_orders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.sale_subscriptions ALTER COLUMN id SET DEFAULT nextval('public.sale_subscriptions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.subscription_plans ALTER COLUMN id SET DEFAULT nextval('public.subscription_plans_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.website_menus ALTER COLUMN id SET DEFAULT nextval('public.website_menus_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.website_pages ALTER COLUMN id SET DEFAULT nextval('public.website_pages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.website_sites ALTER COLUMN id SET DEFAULT nextval('public.website_sites_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_order_groups ALTER COLUMN id SET DEFAULT nextval('public.purchase_order_groups_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_order_lines ALTER COLUMN id SET DEFAULT nextval('public.purchase_order_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_orders ALTER COLUMN id SET DEFAULT nextval('public.purchase_orders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_supplier_infos ALTER COLUMN id SET DEFAULT nextval('public.purchase_supplier_infos_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_requisition_lines ALTER COLUMN id SET DEFAULT nextval('public.purchase_requisition_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.purchase_requisitions ALTER COLUMN id SET DEFAULT nextval('public.purchase_requisitions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.crm_leads ALTER COLUMN id SET DEFAULT nextval('public.crm_leads_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.crm_lost_reasons ALTER COLUMN id SET DEFAULT nextval('public.crm_lost_reasons_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.crm_stages ALTER COLUMN id SET DEFAULT nextval('public.crm_stages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.crm_tags ALTER COLUMN id SET DEFAULT nextval('public.crm_tags_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.edi_certificates ALTER COLUMN id SET DEFAULT nextval('public.edi_certificates_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.edi_documents ALTER COLUMN id SET DEFAULT nextval('public.edi_documents_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.edi_zatca_submissions ALTER COLUMN id SET DEFAULT nextval('public.edi_zatca_submissions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ir_attachments ALTER COLUMN id SET DEFAULT nextval('public.ir_attachments_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_companies ALTER COLUMN id SET DEFAULT nextval('public.res_companies_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_partners ALTER COLUMN id SET DEFAULT nextval('public.res_partners_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.appointment_bookings ALTER COLUMN id SET DEFAULT nextval('public.appointment_bookings_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.appointment_slots ALTER COLUMN id SET DEFAULT nextval('public.appointment_slots_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.appointment_types ALTER COLUMN id SET DEFAULT nextval('public.appointment_types_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.calendar_attendees ALTER COLUMN id SET DEFAULT nextval('public.calendar_attendees_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.calendar_event_alarms ALTER COLUMN id SET DEFAULT nextval('public.calendar_event_alarms_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.calendar_events ALTER COLUMN id SET DEFAULT nextval('public.calendar_events_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.calendar_recurrences ALTER COLUMN id SET DEFAULT nextval('public.calendar_recurrences_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_attendance ALTER COLUMN id SET DEFAULT nextval('public.hr_attendance_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_departments ALTER COLUMN id SET DEFAULT nextval('public.hr_departments_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_employees ALTER COLUMN id SET DEFAULT nextval('public.hr_employees_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_expenses ALTER COLUMN id SET DEFAULT nextval('public.hr_expenses_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_jobs ALTER COLUMN id SET DEFAULT nextval('public.hr_jobs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_leave_allocations ALTER COLUMN id SET DEFAULT nextval('public.hr_leave_allocations_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_leave_requests ALTER COLUMN id SET DEFAULT nextval('public.hr_leave_requests_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_overtime_lines ALTER COLUMN id SET DEFAULT nextval('public.hr_overtime_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_overtime_rules ALTER COLUMN id SET DEFAULT nextval('public.hr_overtime_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.hr_work_entries ALTER COLUMN id SET DEFAULT nextval('public.hr_work_entries_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.planning_roles ALTER COLUMN id SET DEFAULT nextval('public.planning_roles_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.planning_shifts ALTER COLUMN id SET DEFAULT nextval('public.planning_shifts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.recruitment_applicants ALTER COLUMN id SET DEFAULT nextval('public.recruitment_applicants_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.recruitment_interviews ALTER COLUMN id SET DEFAULT nextval('public.recruitment_interviews_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.recruitment_stages ALTER COLUMN id SET DEFAULT nextval('public.recruitment_stages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.resource_calendars ALTER COLUMN id SET DEFAULT nextval('public.resource_calendars_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_bom_lines ALTER COLUMN id SET DEFAULT nextval('public.mrp_bom_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_boms ALTER COLUMN id SET DEFAULT nextval('public.mrp_boms_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_capacity_slots ALTER COLUMN id SET DEFAULT nextval('public.mrp_capacity_slots_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_productions ALTER COLUMN id SET DEFAULT nextval('public.mrp_productions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_productivity_losses ALTER COLUMN id SET DEFAULT nextval('public.mrp_productivity_losses_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_quality_checks ALTER COLUMN id SET DEFAULT nextval('public.mrp_quality_checks_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_quality_points ALTER COLUMN id SET DEFAULT nextval('public.mrp_quality_points_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_routing_operations ALTER COLUMN id SET DEFAULT nextval('public.mrp_routing_operations_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_subcontracting_bom ALTER COLUMN id SET DEFAULT nextval('public.mrp_subcontracting_bom_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_subcontracting_orders ALTER COLUMN id SET DEFAULT nextval('public.mrp_subcontracting_orders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_unbuilds ALTER COLUMN id SET DEFAULT nextval('public.mrp_unbuilds_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_workcenter_calendars ALTER COLUMN id SET DEFAULT nextval('public.mrp_workcenter_calendars_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_workcenter_productivity ALTER COLUMN id SET DEFAULT nextval('public.mrp_workcenter_productivity_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_workcenters ALTER COLUMN id SET DEFAULT nextval('public.mrp_workcenters_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_workorder_time_logs ALTER COLUMN id SET DEFAULT nextval('public.mrp_workorder_time_logs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mrp_workorders ALTER COLUMN id SET DEFAULT nextval('public.mrp_workorders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.portal_users ALTER COLUMN id SET DEFAULT nextval('public.portal_users_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_group_permissions ALTER COLUMN id SET DEFAULT nextval('public.res_group_permissions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_groups ALTER COLUMN id SET DEFAULT nextval('public.res_groups_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_record_rules ALTER COLUMN id SET DEFAULT nextval('public.res_record_rules_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_users ALTER COLUMN id SET DEFAULT nextval('public.res_users_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_analytic_account ALTER COLUMN id SET DEFAULT nextval('public.account_analytic_account_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_analytic_applicability ALTER COLUMN id SET DEFAULT nextval('public.account_analytic_applicability_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_analytic_distribution_model ALTER COLUMN id SET DEFAULT nextval('public.account_analytic_distribution_model_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_analytic_line ALTER COLUMN id SET DEFAULT nextval('public.account_analytic_line_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.account_analytic_plan ALTER COLUMN id SET DEFAULT nextval('public.account_analytic_plan_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_service_types ALTER COLUMN id SET DEFAULT nextval('public.fleet_service_types_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_assignation_logs ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_assignation_logs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_brands ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_brands_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_log_contracts ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_log_contracts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_log_services ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_log_services_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_model_categories ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_model_categories_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_models ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_models_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_odometers ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_odometers_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_states ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_states_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicle_tags ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicle_tags_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.fleet_vehicles ALTER COLUMN id SET DEFAULT nextval('public.fleet_vehicles_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.livechat_channels ALTER COLUMN id SET DEFAULT nextval('public.livechat_channels_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.livechat_messages ALTER COLUMN id SET DEFAULT nextval('public.livechat_messages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.livechat_sessions ALTER COLUMN id SET DEFAULT nextval('public.livechat_sessions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_cash_movements ALTER COLUMN id SET DEFAULT nextval('public.pos_cash_movements_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_configs ALTER COLUMN id SET DEFAULT nextval('public.pos_configs_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_kitchen_ticket_lines ALTER COLUMN id SET DEFAULT nextval('public.pos_kitchen_ticket_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_kitchen_tickets ALTER COLUMN id SET DEFAULT nextval('public.pos_kitchen_tickets_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_order_lines ALTER COLUMN id SET DEFAULT nextval('public.pos_order_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_orders ALTER COLUMN id SET DEFAULT nextval('public.pos_orders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_payment_methods ALTER COLUMN id SET DEFAULT nextval('public.pos_payment_methods_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_payments ALTER COLUMN id SET DEFAULT nextval('public.pos_payments_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_sessions ALTER COLUMN id SET DEFAULT nextval('public.pos_sessions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.pos_sync_batches ALTER COLUMN id SET DEFAULT nextval('public.pos_sync_batches_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.restaurant_floors ALTER COLUMN id SET DEFAULT nextval('public.restaurant_floors_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.restaurant_tables ALTER COLUMN id SET DEFAULT nextval('public.restaurant_tables_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_milestones ALTER COLUMN id SET DEFAULT nextval('public.project_milestones_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_project_stages ALTER COLUMN id SET DEFAULT nextval('public.project_project_stages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_projects ALTER COLUMN id SET DEFAULT nextval('public.project_projects_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_task_tags ALTER COLUMN id SET DEFAULT nextval('public.project_task_tags_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_task_timers ALTER COLUMN id SET DEFAULT nextval('public.project_task_timers_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_task_types ALTER COLUMN id SET DEFAULT nextval('public.project_task_types_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_tasks ALTER COLUMN id SET DEFAULT nextval('public.project_tasks_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.project_timesheets ALTER COLUMN id SET DEFAULT nextval('public.project_timesheets_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.helpdesk_sla_policies ALTER COLUMN id SET DEFAULT nextval('public.helpdesk_sla_policies_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.helpdesk_stages ALTER COLUMN id SET DEFAULT nextval('public.helpdesk_stages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.helpdesk_teams ALTER COLUMN id SET DEFAULT nextval('public.helpdesk_teams_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.helpdesk_tickets ALTER COLUMN id SET DEFAULT nextval('public.helpdesk_tickets_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ir_config_parameters ALTER COLUMN id SET DEFAULT nextval('public.ir_config_parameters_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ir_sequences ALTER COLUMN id SET DEFAULT nextval('public.ir_sequences_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.ir_translation ALTER COLUMN id SET DEFAULT nextval('public.ir_translation_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.knowledge_articles ALTER COLUMN id SET DEFAULT nextval('public.knowledge_articles_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.knowledge_categories ALTER COLUMN id SET DEFAULT nextval('public.knowledge_categories_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_activities ALTER COLUMN id SET DEFAULT nextval('public.mail_activities_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_activity_types ALTER COLUMN id SET DEFAULT nextval('public.mail_activity_types_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_email_queue ALTER COLUMN id SET DEFAULT nextval('public.mail_email_queue_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_followers ALTER COLUMN id SET DEFAULT nextval('public.mail_followers_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_message_subtypes ALTER COLUMN id SET DEFAULT nextval('public.mail_message_subtypes_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_messages ALTER COLUMN id SET DEFAULT nextval('public.mail_messages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_notifications ALTER COLUMN id SET DEFAULT nextval('public.mail_notifications_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mail_tracking_values ALTER COLUMN id SET DEFAULT nextval('public.mail_tracking_values_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mailing_contacts ALTER COLUMN id SET DEFAULT nextval('public.mailing_contacts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mailing_lists ALTER COLUMN id SET DEFAULT nextval('public.mailing_lists_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mailing_traces ALTER COLUMN id SET DEFAULT nextval('public.mailing_traces_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.maintenance_equipment ALTER COLUMN id SET DEFAULT nextval('public.maintenance_equipment_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.maintenance_equipment_categories ALTER COLUMN id SET DEFAULT nextval('public.maintenance_equipment_categories_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.maintenance_requests ALTER COLUMN id SET DEFAULT nextval('public.maintenance_requests_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.maintenance_stages ALTER COLUMN id SET DEFAULT nextval('public.maintenance_stages_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.maintenance_teams ALTER COLUMN id SET DEFAULT nextval('public.maintenance_teams_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.marketing_automation_activities ALTER COLUMN id SET DEFAULT nextval('public.marketing_automation_activities_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.marketing_automations ALTER COLUMN id SET DEFAULT nextval('public.marketing_automations_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.marketing_campaigns ALTER COLUMN id SET DEFAULT nextval('public.marketing_campaigns_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.mass_mailings ALTER COLUMN id SET DEFAULT nextval('public.mass_mailings_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.quality_alerts ALTER COLUMN id SET DEFAULT nextval('public.quality_alerts_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.quality_control_points ALTER COLUMN id SET DEFAULT nextval('public.quality_control_points_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.repair_order_lines ALTER COLUMN id SET DEFAULT nextval('public.repair_order_lines_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.repair_orders ALTER COLUMN id SET DEFAULT nextval('public.repair_orders_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.res_lang ALTER COLUMN id SET DEFAULT nextval('public.res_lang_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.survey_questions ALTER COLUMN id SET DEFAULT nextval('public.survey_questions_id_seq'::regclass);

-- column default
ALTER TABLE ONLY public.survey_surveys ALTER COLUMN id SET DEFAULT nextval('public.survey_surveys_id_seq'::regclass);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT account_accounts_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT account_accounts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_bank_statements
    ADD CONSTRAINT account_bank_statements_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_cash_roundings
    ADD CONSTRAINT account_cash_roundings_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_full_reconciles
    ADD CONSTRAINT account_full_reconciles_matching_number_key UNIQUE (matching_number);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_full_reconciles
    ADD CONSTRAINT account_full_reconciles_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_journals
    ADD CONSTRAINT account_journals_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_journals
    ADD CONSTRAINT account_journals_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_moves
    ADD CONSTRAINT account_moves_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_partial_reconciles
    ADD CONSTRAINT account_partial_reconciles_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_payment_reconciliations
    ADD CONSTRAINT account_payment_reconciliations_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_payment_term_lines
    ADD CONSTRAINT account_payment_term_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_payment_terms
    ADD CONSTRAINT account_payment_terms_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_payments
    ADD CONSTRAINT account_payments_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_reconcile_model_lines
    ADD CONSTRAINT account_reconcile_model_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_reconcile_models
    ADD CONSTRAINT account_reconcile_models_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_taxes
    ADD CONSTRAINT account_taxes_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.accounting_periods
    ADD CONSTRAINT accounting_periods_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_provider_configs
    ADD CONSTRAINT payment_provider_configs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_provider_configs
    ADD CONSTRAINT payment_provider_configs_provider_id_key_environment_key UNIQUE (provider_id, key, environment);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_providers
    ADD CONSTRAINT payment_providers_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_providers
    ADD CONSTRAINT uq_payment_provider_code_company UNIQUE (code, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_tokens
    ADD CONSTRAINT payment_tokens_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT uq_payment_tx_reference UNIQUE (reference);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_webhook_logs
    ADD CONSTRAINT payment_webhook_logs_idempotency_key_key UNIQUE (idempotency_key);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.payment_webhook_logs
    ADD CONSTRAINT payment_webhook_logs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_currencies
    ADD CONSTRAINT res_currencies_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_currencies
    ADD CONSTRAINT res_currencies_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_currency_rates
    ADD CONSTRAINT res_currency_rates_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_currency_rates
    ADD CONSTRAINT uq_currency_rate UNIQUE (currency_id, date, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_carrier
    ADD CONSTRAINT delivery_carrier_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_carrier_country_rel
    ADD CONSTRAINT delivery_carrier_country_rel_pkey PRIMARY KEY (carrier_id, country_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_carrier_state_rel
    ADD CONSTRAINT delivery_carrier_state_rel_pkey PRIMARY KEY (carrier_id, state_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_carrier_zip_prefix_rel
    ADD CONSTRAINT delivery_carrier_zip_prefix_rel_pkey PRIMARY KEY (carrier_id, zip_prefix_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_price_rule
    ADD CONSTRAINT delivery_price_rule_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_zip_prefix
    ADD CONSTRAINT delivery_zip_prefix_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.delivery_zip_prefix
    ADD CONSTRAINT delivery_zip_prefix_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_barcode_nomenclatures
    ADD CONSTRAINT stock_barcode_nomenclatures_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_barcode_rules
    ADD CONSTRAINT stock_barcode_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_landed_cost_lines
    ADD CONSTRAINT stock_landed_cost_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_landed_costs
    ADD CONSTRAINT stock_landed_costs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_locations
    ADD CONSTRAINT stock_locations_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_lots
    ADD CONSTRAINT stock_lots_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_lots
    ADD CONSTRAINT stock_lots_product_name_unique UNIQUE (product_id, name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_orderpoints
    ADD CONSTRAINT stock_orderpoints_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_package_types
    ADD CONSTRAINT stock_package_types_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_packages
    ADD CONSTRAINT stock_packages_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_packages
    ADD CONSTRAINT stock_packages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_procurement_groups
    ADD CONSTRAINT stock_procurement_groups_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_putaway_rules
    ADD CONSTRAINT stock_putaway_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT stock_quants_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT uq_stock_quants_product_location UNIQUE (product_id, location_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_routes
    ADD CONSTRAINT stock_routes_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_rules
    ADD CONSTRAINT stock_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_storage_categories
    ADD CONSTRAINT stock_storage_categories_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_storage_category_capacities
    ADD CONSTRAINT stock_storage_category_capacities_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_valuation_adjustment_lines
    ADD CONSTRAINT stock_valuation_adjustment_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT stock_warehouses_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT stock_warehouses_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_attribute_values
    ADD CONSTRAINT product_attribute_values_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_attributes
    ADD CONSTRAINT product_attributes_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_packagings
    ADD CONSTRAINT product_packagings_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_pricelist_items
    ADD CONSTRAINT product_pricelist_items_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_pricelists
    ADD CONSTRAINT product_pricelists_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_values
    ADD CONSTRAINT product_values_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_variant_attributes
    ADD CONSTRAINT product_variant_attributes_pkey PRIMARY KEY (variant_id, attribute_value_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.product_variants
    ADD CONSTRAINT product_variants_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.uom_uoms
    ADD CONSTRAINT uom_uoms_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ecommerce_cart_lines
    ADD CONSTRAINT ecommerce_cart_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_card_history
    ADD CONSTRAINT loyalty_card_history_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_cards
    ADD CONSTRAINT loyalty_cards_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_cards
    ADD CONSTRAINT loyalty_cards_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_mails
    ADD CONSTRAINT loyalty_mails_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_program_pricelists
    ADD CONSTRAINT loyalty_program_pricelists_pkey PRIMARY KEY (program_id, pricelist_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_programs
    ADD CONSTRAINT loyalty_programs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_rewards
    ADD CONSTRAINT loyalty_rewards_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.loyalty_rules
    ADD CONSTRAINT loyalty_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_order_coupon_points
    ADD CONSTRAINT sale_order_coupon_points_order_id_coupon_id_key UNIQUE (order_id, coupon_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_order_coupon_points
    ADD CONSTRAINT sale_order_coupon_points_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_order_invoices
    ADD CONSTRAINT sale_order_invoices_pkey PRIMARY KEY (order_id, move_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT sale_order_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.website_menus
    ADD CONSTRAINT website_menus_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.website_pages
    ADD CONSTRAINT website_pages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.website_pages
    ADD CONSTRAINT website_pages_website_id_slug_key UNIQUE (website_id, slug);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.website_sites
    ADD CONSTRAINT website_sites_domain_key UNIQUE (domain);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.website_sites
    ADD CONSTRAINT website_sites_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_order_group_members
    ADD CONSTRAINT purchase_order_group_members_order_id_key UNIQUE (order_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_order_group_members
    ADD CONSTRAINT purchase_order_group_members_pkey PRIMARY KEY (group_id, order_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_order_groups
    ADD CONSTRAINT purchase_order_groups_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_order_invoices
    ADD CONSTRAINT purchase_order_invoices_pkey PRIMARY KEY (order_id, move_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_order_lines
    ADD CONSTRAINT purchase_order_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_requisition_line_id_key UNIQUE (requisition_line_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_lead_tags
    ADD CONSTRAINT crm_lead_tags_pkey PRIMARY KEY (lead_id, tag_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_leads
    ADD CONSTRAINT crm_leads_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_lost_reasons
    ADD CONSTRAINT crm_lost_reasons_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_stages
    ADD CONSTRAINT crm_stages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_tags
    ADD CONSTRAINT crm_tags_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.crm_tags
    ADD CONSTRAINT crm_tags_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.edi_certificates
    ADD CONSTRAINT edi_certificates_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.edi_documents
    ADD CONSTRAINT edi_documents_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.edi_zatca_submissions
    ADD CONSTRAINT edi_zatca_submissions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_attachments
    ADD CONSTRAINT ir_attachments_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_companies
    ADD CONSTRAINT res_companies_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_partners
    ADD CONSTRAINT res_partners_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.appointment_bookings
    ADD CONSTRAINT appointment_bookings_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.appointment_slots
    ADD CONSTRAINT appointment_slots_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.appointment_type_users
    ADD CONSTRAINT appointment_type_users_pkey PRIMARY KEY (appointment_type_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.appointment_types
    ADD CONSTRAINT appointment_types_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.appointment_types
    ADD CONSTRAINT appointment_types_slug_key UNIQUE (slug);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.calendar_attendees
    ADD CONSTRAINT calendar_attendees_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.calendar_attendees
    ADD CONSTRAINT calendar_attendees_token_key UNIQUE (token);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.calendar_event_alarms
    ADD CONSTRAINT calendar_event_alarms_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.calendar_recurrences
    ADD CONSTRAINT calendar_recurrences_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_attendance
    ADD CONSTRAINT hr_attendance_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_departments
    ADD CONSTRAINT hr_departments_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_expense_taxes
    ADD CONSTRAINT hr_expense_taxes_pkey PRIMARY KEY (expense_id, tax_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_jobs
    ADD CONSTRAINT hr_jobs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_leave_allocations
    ADD CONSTRAINT hr_leave_allocations_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_leave_requests
    ADD CONSTRAINT hr_leave_requests_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_overtime_lines
    ADD CONSTRAINT hr_overtime_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_overtime_rules
    ADD CONSTRAINT hr_overtime_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.hr_work_entries
    ADD CONSTRAINT hr_work_entries_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.planning_roles
    ADD CONSTRAINT planning_roles_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.planning_shifts
    ADD CONSTRAINT planning_shifts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.recruitment_interviews
    ADD CONSTRAINT recruitment_interviews_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.recruitment_stages
    ADD CONSTRAINT recruitment_stages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.resource_calendars
    ADD CONSTRAINT resource_calendars_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_bom_lines
    ADD CONSTRAINT mrp_bom_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_boms
    ADD CONSTRAINT mrp_boms_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_capacity_slots
    ADD CONSTRAINT mrp_capacity_slots_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_productions
    ADD CONSTRAINT mrp_productions_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_productions
    ADD CONSTRAINT mrp_productions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_productivity_losses
    ADD CONSTRAINT mrp_productivity_losses_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_quality_checks
    ADD CONSTRAINT mrp_quality_checks_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_quality_points
    ADD CONSTRAINT mrp_quality_points_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_routing_operations
    ADD CONSTRAINT mrp_routing_operations_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_subcontracting_bom
    ADD CONSTRAINT mrp_subcontracting_bom_bom_id_subcontractor_id_key UNIQUE (bom_id, subcontractor_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_subcontracting_bom
    ADD CONSTRAINT mrp_subcontracting_bom_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_subcontracting_orders
    ADD CONSTRAINT mrp_subcontracting_orders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workcenter_calendars
    ADD CONSTRAINT mrp_workcenter_calendars_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workcenter_calendars
    ADD CONSTRAINT mrp_workcenter_calendars_workcenter_id_day_of_week_hour_fro_key UNIQUE (workcenter_id, day_of_week, hour_from);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workcenter_productivity
    ADD CONSTRAINT mrp_workcenter_productivity_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workcenters
    ADD CONSTRAINT mrp_workcenters_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workorder_time_logs
    ADD CONSTRAINT mrp_workorder_time_logs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mrp_workorders
    ADD CONSTRAINT mrp_workorders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.portal_users
    ADD CONSTRAINT portal_users_email_key UNIQUE (email);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.portal_users
    ADD CONSTRAINT portal_users_invite_token_key UNIQUE (invite_token);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.portal_users
    ADD CONSTRAINT portal_users_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_group_permissions
    ADD CONSTRAINT res_group_permissions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_group_permissions
    ADD CONSTRAINT uq_res_group_permissions_group_model UNIQUE (group_id, model);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_groups
    ADD CONSTRAINT res_groups_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_groups_implied_rel
    ADD CONSTRAINT res_groups_implied_rel_pkey PRIMARY KEY (group_id, implied_group_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_groups_users_rel
    ADD CONSTRAINT res_groups_users_rel_pkey PRIMARY KEY (group_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_record_rules
    ADD CONSTRAINT res_record_rules_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_users
    ADD CONSTRAINT res_users_login_key UNIQUE (login);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_users
    ADD CONSTRAINT res_users_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_analytic_account
    ADD CONSTRAINT account_analytic_account_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_analytic_applicability
    ADD CONSTRAINT account_analytic_applicability_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_analytic_distribution_model
    ADD CONSTRAINT account_analytic_distribution_model_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_analytic_line
    ADD CONSTRAINT account_analytic_line_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.account_analytic_plan
    ADD CONSTRAINT account_analytic_plan_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_service_types
    ADD CONSTRAINT fleet_service_types_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_assignation_logs
    ADD CONSTRAINT fleet_vehicle_assignation_logs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_brands
    ADD CONSTRAINT fleet_vehicle_brands_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_log_contracts
    ADD CONSTRAINT fleet_vehicle_log_contracts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_log_services
    ADD CONSTRAINT fleet_vehicle_log_services_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_model_categories
    ADD CONSTRAINT fleet_vehicle_model_categories_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_models
    ADD CONSTRAINT fleet_vehicle_models_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_odometers
    ADD CONSTRAINT fleet_vehicle_odometers_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_states
    ADD CONSTRAINT fleet_vehicle_states_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_tag_rel
    ADD CONSTRAINT fleet_vehicle_tag_rel_pkey PRIMARY KEY (vehicle_id, tag_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicle_tags
    ADD CONSTRAINT fleet_vehicle_tags_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_license_plate_key UNIQUE (license_plate);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.livechat_channel_users
    ADD CONSTRAINT livechat_channel_users_pkey PRIMARY KEY (channel_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.livechat_channels
    ADD CONSTRAINT livechat_channels_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.livechat_messages
    ADD CONSTRAINT livechat_messages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.livechat_sessions
    ADD CONSTRAINT livechat_sessions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_cash_movements
    ADD CONSTRAINT pos_cash_movements_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_config_payment_method_rel
    ADD CONSTRAINT pos_config_payment_method_rel_pkey PRIMARY KEY (config_id, payment_method_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_kitchen_ticket_lines
    ADD CONSTRAINT pos_kitchen_ticket_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_kitchen_tickets
    ADD CONSTRAINT pos_kitchen_tickets_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_order_lines
    ADD CONSTRAINT pos_order_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_client_uuid_key UNIQUE (client_uuid);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_payment_methods
    ADD CONSTRAINT pos_payment_methods_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_payments
    ADD CONSTRAINT pos_payments_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_sync_batches
    ADD CONSTRAINT pos_sync_batches_idempotency_key_key UNIQUE (idempotency_key);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.pos_sync_batches
    ADD CONSTRAINT pos_sync_batches_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.restaurant_floors
    ADD CONSTRAINT restaurant_floors_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.restaurant_tables
    ADD CONSTRAINT restaurant_tables_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_milestones
    ADD CONSTRAINT project_milestones_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_project_stages
    ADD CONSTRAINT project_project_stages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_project_stages
    ADD CONSTRAINT uq_project_project_stages_name_company UNIQUE (name, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_assignees
    ADD CONSTRAINT project_task_assignees_pkey PRIMARY KEY (task_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_dependencies
    ADD CONSTRAINT project_task_dependencies_pkey PRIMARY KEY (task_id, depends_on_task_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_tag_rel
    ADD CONSTRAINT project_task_tag_rel_pkey PRIMARY KEY (task_id, tag_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_tags
    ADD CONSTRAINT project_task_tags_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_tags
    ADD CONSTRAINT uq_project_task_tags_name UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_timers
    ADD CONSTRAINT project_task_timers_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_type_projects
    ADD CONSTRAINT project_task_type_projects_pkey PRIMARY KEY (task_type_id, project_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_types
    ADD CONSTRAINT project_task_types_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_task_types
    ADD CONSTRAINT uq_project_task_types_name_company UNIQUE (name, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.helpdesk_sla_policies
    ADD CONSTRAINT helpdesk_sla_policies_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.helpdesk_stages
    ADD CONSTRAINT helpdesk_stages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.helpdesk_teams
    ADD CONSTRAINT helpdesk_teams_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_number_key UNIQUE (number);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_config_parameters
    ADD CONSTRAINT ir_config_parameters_key_key UNIQUE (key);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_config_parameters
    ADD CONSTRAINT ir_config_parameters_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_sequences
    ADD CONSTRAINT ir_sequences_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_sequences
    ADD CONSTRAINT ir_sequences_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.ir_translation
    ADD CONSTRAINT ir_translation_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.knowledge_articles
    ADD CONSTRAINT knowledge_articles_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.knowledge_articles
    ADD CONSTRAINT knowledge_articles_slug_key UNIQUE (slug);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.knowledge_categories
    ADD CONSTRAINT knowledge_categories_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_activity_types
    ADD CONSTRAINT mail_activity_types_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_activity_types
    ADD CONSTRAINT uq_mail_activity_types_name_company UNIQUE (name, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_email_queue
    ADD CONSTRAINT mail_email_queue_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT mail_followers_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT uq_mail_followers_res_partner UNIQUE (res_model, res_id, partner_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT uq_mail_followers_res_user UNIQUE (res_model, res_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_followers_subtypes_rel
    ADD CONSTRAINT mail_followers_subtypes_rel_pkey PRIMARY KEY (follower_id, subtype_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_message_subtypes
    ADD CONSTRAINT mail_message_subtypes_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_message_subtypes
    ADD CONSTRAINT uq_mail_message_subtypes_name_model UNIQUE (name, res_model);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_notifications
    ADD CONSTRAINT mail_notifications_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mail_tracking_values
    ADD CONSTRAINT mail_tracking_values_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_contacts
    ADD CONSTRAINT mailing_contacts_email_company_id_key UNIQUE (email, company_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_contacts
    ADD CONSTRAINT mailing_contacts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_list_contact_rel
    ADD CONSTRAINT mailing_list_contact_rel_pkey PRIMARY KEY (list_id, contact_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_lists
    ADD CONSTRAINT mailing_lists_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_traces
    ADD CONSTRAINT mailing_traces_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mailing_traces
    ADD CONSTRAINT mailing_traces_tracking_code_key UNIQUE (tracking_code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_equipment_categories
    ADD CONSTRAINT maintenance_equipment_categories_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_stages
    ADD CONSTRAINT maintenance_stages_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_team_members
    ADD CONSTRAINT maintenance_team_members_pkey PRIMARY KEY (team_id, user_id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.maintenance_teams
    ADD CONSTRAINT maintenance_teams_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.marketing_automation_activities
    ADD CONSTRAINT marketing_automation_activities_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.marketing_automations
    ADD CONSTRAINT marketing_automations_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.marketing_campaigns
    ADD CONSTRAINT marketing_campaigns_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.mass_mailings
    ADD CONSTRAINT mass_mailings_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.quality_alerts
    ADD CONSTRAINT quality_alerts_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.quality_control_points
    ADD CONSTRAINT quality_control_points_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.repair_order_lines
    ADD CONSTRAINT repair_order_lines_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_name_key UNIQUE (name);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_lang
    ADD CONSTRAINT res_lang_code_key UNIQUE (code);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.res_lang
    ADD CONSTRAINT res_lang_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.survey_questions
    ADD CONSTRAINT survey_questions_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.survey_surveys
    ADD CONSTRAINT survey_surveys_pkey PRIMARY KEY (id);

-- PK/UNIQUE/CHECK
ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);

-- foreign key
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT account_accounts_account_stock_expense_id_fkey FOREIGN KEY (account_stock_expense_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT account_accounts_account_stock_variation_id_fkey FOREIGN KEY (account_stock_variation_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT account_accounts_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_accounts
    ADD CONSTRAINT fk_account_accounts_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_bank_statement_lines
    ADD CONSTRAINT account_bank_statement_lines_statement_id_fkey FOREIGN KEY (statement_id) REFERENCES public.account_bank_statements(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_bank_statements
    ADD CONSTRAINT account_bank_statements_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_bank_statements
    ADD CONSTRAINT account_bank_statements_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_cash_roundings
    ADD CONSTRAINT account_cash_roundings_loss_account_id_fkey FOREIGN KEY (loss_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_cash_roundings
    ADD CONSTRAINT account_cash_roundings_profit_account_id_fkey FOREIGN KEY (profit_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_full_reconciles
    ADD CONSTRAINT account_full_reconciles_exchange_move_id_fkey FOREIGN KEY (exchange_move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_journals
    ADD CONSTRAINT account_journals_default_account_id_fkey FOREIGN KEY (default_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_journals
    ADD CONSTRAINT account_journals_suspense_account_id_fkey FOREIGN KEY (suspense_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_cogs_origin_id_fkey FOREIGN KEY (cogs_origin_id) REFERENCES public.account_move_lines(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.account_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_move_lines
    ADD CONSTRAINT account_move_lines_statement_line_id_fkey FOREIGN KEY (statement_line_id) REFERENCES public.account_bank_statement_lines(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_moves
    ADD CONSTRAINT account_moves_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_moves
    ADD CONSTRAINT account_moves_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_moves
    ADD CONSTRAINT account_moves_payment_term_id_fkey FOREIGN KEY (payment_term_id) REFERENCES public.account_payment_terms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_moves
    ADD CONSTRAINT account_moves_reversed_entry_id_fkey FOREIGN KEY (reversed_entry_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_partial_reconciles
    ADD CONSTRAINT account_partial_reconciles_credit_line_id_fkey FOREIGN KEY (credit_line_id) REFERENCES public.account_move_lines(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_partial_reconciles
    ADD CONSTRAINT account_partial_reconciles_credit_move_id_fkey FOREIGN KEY (credit_move_id) REFERENCES public.account_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_partial_reconciles
    ADD CONSTRAINT account_partial_reconciles_debit_line_id_fkey FOREIGN KEY (debit_line_id) REFERENCES public.account_move_lines(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_partial_reconciles
    ADD CONSTRAINT account_partial_reconciles_debit_move_id_fkey FOREIGN KEY (debit_move_id) REFERENCES public.account_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_payment_reconciliations
    ADD CONSTRAINT account_payment_reconciliations_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES public.account_moves(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_payment_reconciliations
    ADD CONSTRAINT account_payment_reconciliations_payment_id_fkey FOREIGN KEY (payment_id) REFERENCES public.account_payments(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_payment_term_lines
    ADD CONSTRAINT account_payment_term_lines_payment_term_id_fkey FOREIGN KEY (payment_term_id) REFERENCES public.account_payment_terms(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_payments
    ADD CONSTRAINT account_payments_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_payments
    ADD CONSTRAINT account_payments_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_payments
    ADD CONSTRAINT account_payments_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_payments
    ADD CONSTRAINT fk_account_payments_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_reconcile_model_lines
    ADD CONSTRAINT account_reconcile_model_lines_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_reconcile_model_lines
    ADD CONSTRAINT account_reconcile_model_lines_reconcile_model_id_fkey FOREIGN KEY (reconcile_model_id) REFERENCES public.account_reconcile_models(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_reconcile_models
    ADD CONSTRAINT account_reconcile_models_mapped_partner_id_fkey FOREIGN KEY (mapped_partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_taxes
    ADD CONSTRAINT account_taxes_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_taxes
    ADD CONSTRAINT account_taxes_refund_account_id_fkey FOREIGN KEY (refund_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.accounting_periods
    ADD CONSTRAINT accounting_periods_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.accounting_periods
    ADD CONSTRAINT accounting_periods_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.payment_provider_configs
    ADD CONSTRAINT payment_provider_configs_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.payment_providers(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.payment_providers
    ADD CONSTRAINT payment_providers_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_original_transaction_id_fkey FOREIGN KEY (original_transaction_id) REFERENCES public.payment_transactions(id);

-- foreign key
ALTER TABLE ONLY public.payment_refunds
    ADD CONSTRAINT payment_refunds_refund_transaction_id_fkey FOREIGN KEY (refund_transaction_id) REFERENCES public.payment_transactions(id);

-- foreign key
ALTER TABLE ONLY public.payment_tokens
    ADD CONSTRAINT payment_tokens_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.payment_providers(id);

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_payment_id_fkey FOREIGN KEY (payment_id) REFERENCES public.account_payments(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.payment_providers(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.payment_transactions
    ADD CONSTRAINT payment_transactions_sale_order_id_fkey FOREIGN KEY (sale_order_id) REFERENCES public.sale_orders(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.res_currency_rates
    ADD CONSTRAINT res_currency_rates_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.res_currencies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.delivery_carrier
    ADD CONSTRAINT delivery_carrier_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.delivery_carrier
    ADD CONSTRAINT delivery_carrier_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.delivery_carrier_country_rel
    ADD CONSTRAINT delivery_carrier_country_rel_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.delivery_carrier_state_rel
    ADD CONSTRAINT delivery_carrier_state_rel_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.delivery_carrier_zip_prefix_rel
    ADD CONSTRAINT delivery_carrier_zip_prefix_rel_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.delivery_carrier_zip_prefix_rel
    ADD CONSTRAINT delivery_carrier_zip_prefix_rel_zip_prefix_id_fkey FOREIGN KEY (zip_prefix_id) REFERENCES public.delivery_zip_prefix(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.delivery_price_rule
    ADD CONSTRAINT delivery_price_rule_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_price_diff_account_id_fkey FOREIGN KEY (price_diff_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_product_category_id_fkey FOREIGN KEY (product_category_id) REFERENCES public.product_categories(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_stock_input_account_id_fkey FOREIGN KEY (stock_input_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_stock_journal_id_fkey FOREIGN KEY (stock_journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_stock_output_account_id_fkey FOREIGN KEY (stock_output_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_account_config
    ADD CONSTRAINT stock_account_config_stock_valuation_account_id_fkey FOREIGN KEY (stock_valuation_account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_barcode_rules
    ADD CONSTRAINT stock_barcode_rules_nomenclature_id_fkey FOREIGN KEY (nomenclature_id) REFERENCES public.stock_barcode_nomenclatures(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_landed_cost_lines
    ADD CONSTRAINT stock_landed_cost_lines_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_landed_cost_lines
    ADD CONSTRAINT stock_landed_cost_lines_landed_cost_id_fkey FOREIGN KEY (landed_cost_id) REFERENCES public.stock_landed_costs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_landed_cost_lines
    ADD CONSTRAINT stock_landed_cost_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_landed_costs
    ADD CONSTRAINT stock_landed_costs_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_landed_costs
    ADD CONSTRAINT stock_landed_costs_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_landed_costs
    ADD CONSTRAINT stock_landed_costs_vendor_bill_id_fkey FOREIGN KEY (vendor_bill_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_locations
    ADD CONSTRAINT fk_stock_locations_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_locations
    ADD CONSTRAINT stock_locations_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_locations
    ADD CONSTRAINT stock_locations_valuation_account_id_fkey FOREIGN KEY (valuation_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_lots
    ADD CONSTRAINT stock_lots_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.stock_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_move_lines
    ADD CONSTRAINT stock_move_lines_result_package_id_fkey FOREIGN KEY (result_package_id) REFERENCES public.stock_packages(id);

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_picking_id_fkey FOREIGN KEY (picking_id) REFERENCES public.stock_pickings(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_procurement_group_id_fkey FOREIGN KEY (procurement_group_id) REFERENCES public.stock_procurement_groups(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_production_finished_id_fkey FOREIGN KEY (production_finished_id) REFERENCES public.mrp_productions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_production_id_fkey FOREIGN KEY (production_id) REFERENCES public.mrp_productions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_purchase_line_id_fkey FOREIGN KEY (purchase_line_id) REFERENCES public.purchase_order_lines(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_moves
    ADD CONSTRAINT stock_moves_sale_line_id_fkey FOREIGN KEY (sale_line_id) REFERENCES public.sale_order_lines(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_orderpoints
    ADD CONSTRAINT stock_orderpoints_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_orderpoints
    ADD CONSTRAINT stock_orderpoints_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_orderpoints
    ADD CONSTRAINT stock_orderpoints_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_orderpoints
    ADD CONSTRAINT stock_orderpoints_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.stock_warehouses(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_packages
    ADD CONSTRAINT stock_packages_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.stock_packages
    ADD CONSTRAINT stock_packages_package_type_id_fkey FOREIGN KEY (package_type_id) REFERENCES public.stock_package_types(id);

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT fk_stock_pickings_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_backorder_of_id_fkey FOREIGN KEY (backorder_of_id) REFERENCES public.stock_pickings(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id);

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_pickings
    ADD CONSTRAINT stock_pickings_procurement_group_id_fkey FOREIGN KEY (procurement_group_id) REFERENCES public.stock_procurement_groups(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_putaway_rules
    ADD CONSTRAINT stock_putaway_rules_location_in_id_fkey FOREIGN KEY (location_in_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.stock_putaway_rules
    ADD CONSTRAINT stock_putaway_rules_location_out_id_fkey FOREIGN KEY (location_out_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.stock_putaway_rules
    ADD CONSTRAINT stock_putaway_rules_storage_category_id_fkey FOREIGN KEY (storage_category_id) REFERENCES public.stock_storage_categories(id);

-- foreign key
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT fk_stock_quants_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT stock_quants_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT stock_quants_package_id_fkey FOREIGN KEY (package_id) REFERENCES public.stock_packages(id);

-- foreign key
ALTER TABLE ONLY public.stock_quants
    ADD CONSTRAINT stock_quants_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_rules
    ADD CONSTRAINT stock_rules_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.stock_rules
    ADD CONSTRAINT stock_rules_route_id_fkey FOREIGN KEY (route_id) REFERENCES public.stock_routes(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_storage_category_capacities
    ADD CONSTRAINT stock_storage_category_capacities_package_type_id_fkey FOREIGN KEY (package_type_id) REFERENCES public.stock_package_types(id);

-- foreign key
ALTER TABLE ONLY public.stock_storage_category_capacities
    ADD CONSTRAINT stock_storage_category_capacities_storage_category_id_fkey FOREIGN KEY (storage_category_id) REFERENCES public.stock_storage_categories(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_valuation_adjustment_lines
    ADD CONSTRAINT stock_valuation_adjustment_lines_cost_line_id_fkey FOREIGN KEY (cost_line_id) REFERENCES public.stock_landed_cost_lines(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_valuation_adjustment_lines
    ADD CONSTRAINT stock_valuation_adjustment_lines_landed_cost_id_fkey FOREIGN KEY (landed_cost_id) REFERENCES public.stock_landed_costs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_valuation_adjustment_lines
    ADD CONSTRAINT stock_valuation_adjustment_lines_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.stock_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_valuation_adjustment_lines
    ADD CONSTRAINT stock_valuation_adjustment_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT fk_stock_warehouses_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT stock_warehouses_lot_stock_id_fkey FOREIGN KEY (lot_stock_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT stock_warehouses_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.stock_warehouses
    ADD CONSTRAINT stock_warehouses_view_location_id_fkey FOREIGN KEY (view_location_id) REFERENCES public.stock_locations(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.product_attribute_values
    ADD CONSTRAINT product_attribute_values_attribute_id_fkey FOREIGN KEY (attribute_id) REFERENCES public.product_attributes(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.product_categories(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_property_price_difference_account_id_fkey FOREIGN KEY (property_price_difference_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_property_stock_journal_id_fkey FOREIGN KEY (property_stock_journal_id) REFERENCES public.account_journals(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_categories
    ADD CONSTRAINT product_categories_property_stock_valuation_account_id_fkey FOREIGN KEY (property_stock_valuation_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_packagings
    ADD CONSTRAINT product_packagings_package_type_id_fkey FOREIGN KEY (package_type_id) REFERENCES public.stock_package_types(id);

-- foreign key
ALTER TABLE ONLY public.product_pricelist_items
    ADD CONSTRAINT product_pricelist_items_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.product_categories(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_pricelist_items
    ADD CONSTRAINT product_pricelist_items_pricelist_id_fkey FOREIGN KEY (pricelist_id) REFERENCES public.product_pricelists(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_pricelist_items
    ADD CONSTRAINT product_pricelist_items_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.product_templates(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_pricelist_items
    ADD CONSTRAINT product_pricelist_items_variant_id_fkey FOREIGN KEY (variant_id) REFERENCES public.product_variants(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT fk_product_templates_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.product_categories(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_price_difference_account_id_fkey FOREIGN KEY (price_difference_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_stock_journal_id_fkey FOREIGN KEY (stock_journal_id) REFERENCES public.account_journals(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_stock_valuation_account_id_fkey FOREIGN KEY (stock_valuation_account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_templates
    ADD CONSTRAINT product_templates_uom_id_fkey FOREIGN KEY (uom_id) REFERENCES public.uom_uoms(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.product_values
    ADD CONSTRAINT product_values_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.stock_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.product_values
    ADD CONSTRAINT product_values_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_variant_attributes
    ADD CONSTRAINT product_variant_attributes_attribute_value_id_fkey FOREIGN KEY (attribute_value_id) REFERENCES public.product_attribute_values(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_variant_attributes
    ADD CONSTRAINT product_variant_attributes_variant_id_fkey FOREIGN KEY (variant_id) REFERENCES public.product_variants(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.product_variants
    ADD CONSTRAINT product_variants_template_id_fkey FOREIGN KEY (template_id) REFERENCES public.product_templates(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.ecommerce_cart_lines
    ADD CONSTRAINT ecommerce_cart_lines_cart_id_fkey FOREIGN KEY (cart_id) REFERENCES public.ecommerce_carts(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.ecommerce_cart_lines
    ADD CONSTRAINT ecommerce_cart_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_converted_order_id_fkey FOREIGN KEY (converted_order_id) REFERENCES public.sale_orders(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_delivery_method_id_fkey FOREIGN KEY (delivery_method_id) REFERENCES public.delivery_carrier(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_invoice_address_id_fkey FOREIGN KEY (invoice_address_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_pricelist_id_fkey FOREIGN KEY (pricelist_id) REFERENCES public.product_pricelists(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_shipping_address_id_fkey FOREIGN KEY (shipping_address_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.ecommerce_carts
    ADD CONSTRAINT ecommerce_carts_website_id_fkey FOREIGN KEY (website_id) REFERENCES public.website_sites(id);

-- foreign key
ALTER TABLE ONLY public.loyalty_card_history
    ADD CONSTRAINT loyalty_card_history_card_id_fkey FOREIGN KEY (card_id) REFERENCES public.loyalty_cards(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.loyalty_cards
    ADD CONSTRAINT loyalty_cards_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.loyalty_mails
    ADD CONSTRAINT loyalty_mails_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.loyalty_program_pricelists
    ADD CONSTRAINT loyalty_program_pricelists_pricelist_id_fkey FOREIGN KEY (pricelist_id) REFERENCES public.product_pricelists(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.loyalty_program_pricelists
    ADD CONSTRAINT loyalty_program_pricelists_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.loyalty_rewards
    ADD CONSTRAINT loyalty_rewards_discount_line_product_id_fkey FOREIGN KEY (discount_line_product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.loyalty_rewards
    ADD CONSTRAINT loyalty_rewards_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.loyalty_rewards
    ADD CONSTRAINT loyalty_rewards_reward_product_id_fkey FOREIGN KEY (reward_product_id) REFERENCES public.product_templates(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.loyalty_rules
    ADD CONSTRAINT loyalty_rules_program_id_fkey FOREIGN KEY (program_id) REFERENCES public.loyalty_programs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_coupon_points
    ADD CONSTRAINT sale_order_coupon_points_coupon_id_fkey FOREIGN KEY (coupon_id) REFERENCES public.loyalty_cards(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_coupon_points
    ADD CONSTRAINT sale_order_coupon_points_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.sale_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_invoices
    ADD CONSTRAINT sale_order_invoices_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.account_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_invoices
    ADD CONSTRAINT sale_order_invoices_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.sale_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT fk_sale_order_lines_coupon FOREIGN KEY (coupon_id) REFERENCES public.loyalty_cards(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT fk_sale_order_lines_reward FOREIGN KEY (reward_id) REFERENCES public.loyalty_rewards(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT sale_order_lines_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.sale_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT sale_order_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.sale_order_lines
    ADD CONSTRAINT sale_order_lines_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT fk_sale_orders_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES public.delivery_carrier(id);

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_payment_term_id_fkey FOREIGN KEY (payment_term_id) REFERENCES public.account_payment_terms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_pricelist_id_fkey FOREIGN KEY (pricelist_id) REFERENCES public.product_pricelists(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.sale_orders
    ADD CONSTRAINT sale_orders_procurement_group_id_fkey FOREIGN KEY (procurement_group_id) REFERENCES public.stock_procurement_groups(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_payment_token_id_fkey FOREIGN KEY (payment_token_id) REFERENCES public.payment_tokens(id);

-- foreign key
ALTER TABLE ONLY public.sale_subscriptions
    ADD CONSTRAINT sale_subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.subscription_plans(id);

-- foreign key
ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.subscription_plans
    ADD CONSTRAINT subscription_plans_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.website_menus
    ADD CONSTRAINT website_menus_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.website_menus(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.website_menus
    ADD CONSTRAINT website_menus_website_id_fkey FOREIGN KEY (website_id) REFERENCES public.website_sites(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.website_pages
    ADD CONSTRAINT website_pages_website_id_fkey FOREIGN KEY (website_id) REFERENCES public.website_sites(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.website_sites
    ADD CONSTRAINT website_sites_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.website_sites
    ADD CONSTRAINT website_sites_pricelist_id_fkey FOREIGN KEY (pricelist_id) REFERENCES public.product_pricelists(id);

-- foreign key
ALTER TABLE ONLY public.website_sites
    ADD CONSTRAINT website_sites_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.stock_warehouses(id);

-- foreign key
ALTER TABLE ONLY public.purchase_order_group_members
    ADD CONSTRAINT purchase_order_group_members_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.purchase_order_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_order_group_members
    ADD CONSTRAINT purchase_order_group_members_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_order_groups
    ADD CONSTRAINT purchase_order_groups_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_order_invoices
    ADD CONSTRAINT purchase_order_invoices_move_id_fkey FOREIGN KEY (move_id) REFERENCES public.account_moves(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_order_invoices
    ADD CONSTRAINT purchase_order_invoices_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_order_lines
    ADD CONSTRAINT purchase_order_lines_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.purchase_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_order_lines
    ADD CONSTRAINT purchase_order_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_order_lines
    ADD CONSTRAINT purchase_order_lines_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT fk_purchase_orders_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_payment_term_id_fkey FOREIGN KEY (payment_term_id) REFERENCES public.account_payment_terms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_procurement_group_id_fkey FOREIGN KEY (procurement_group_id) REFERENCES public.stock_procurement_groups(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_orders
    ADD CONSTRAINT purchase_orders_requisition_id_fkey FOREIGN KEY (requisition_id) REFERENCES public.purchase_requisitions(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.res_currencies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_requisition_id_fkey FOREIGN KEY (requisition_id) REFERENCES public.purchase_requisitions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_requisition_line_id_fkey FOREIGN KEY (requisition_line_id) REFERENCES public.purchase_requisition_lines(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_supplier_infos
    ADD CONSTRAINT purchase_supplier_infos_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_product_uom_fkey FOREIGN KEY (product_uom) REFERENCES public.uom_uoms(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_requisition_id_fkey FOREIGN KEY (requisition_id) REFERENCES public.purchase_requisitions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_supplier_id_fkey FOREIGN KEY (supplier_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.res_currencies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.crm_lead_tags
    ADD CONSTRAINT crm_lead_tags_lead_id_fkey FOREIGN KEY (lead_id) REFERENCES public.crm_leads(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.crm_lead_tags
    ADD CONSTRAINT crm_lead_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.crm_tags(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.crm_leads
    ADD CONSTRAINT crm_leads_lost_reason_id_fkey FOREIGN KEY (lost_reason_id) REFERENCES public.crm_lost_reasons(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.crm_leads
    ADD CONSTRAINT crm_leads_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.crm_leads
    ADD CONSTRAINT crm_leads_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.crm_stages(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.crm_leads
    ADD CONSTRAINT fk_crm_leads_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.crm_stages
    ADD CONSTRAINT fk_crm_stages_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.edi_zatca_submissions
    ADD CONSTRAINT edi_zatca_submissions_edi_document_id_fkey FOREIGN KEY (edi_document_id) REFERENCES public.edi_documents(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_companies
    ADD CONSTRAINT fk_res_companies_partner FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.res_companies
    ADD CONSTRAINT res_companies_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.res_currencies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.res_companies
    ADD CONSTRAINT res_companies_lc_journal_id_fkey FOREIGN KEY (lc_journal_id) REFERENCES public.account_journals(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.res_partners
    ADD CONSTRAINT fk_res_partners_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.res_partners
    ADD CONSTRAINT res_partners_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.appointment_bookings
    ADD CONSTRAINT appointment_bookings_appointment_type_id_fkey FOREIGN KEY (appointment_type_id) REFERENCES public.appointment_types(id);

-- foreign key
ALTER TABLE ONLY public.appointment_bookings
    ADD CONSTRAINT appointment_bookings_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.calendar_events(id);

-- foreign key
ALTER TABLE ONLY public.appointment_bookings
    ADD CONSTRAINT appointment_bookings_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.appointment_slots
    ADD CONSTRAINT appointment_slots_appointment_type_id_fkey FOREIGN KEY (appointment_type_id) REFERENCES public.appointment_types(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.appointment_type_users
    ADD CONSTRAINT appointment_type_users_appointment_type_id_fkey FOREIGN KEY (appointment_type_id) REFERENCES public.appointment_types(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.appointment_type_users
    ADD CONSTRAINT appointment_type_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.appointment_types
    ADD CONSTRAINT appointment_types_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.calendar_attendees
    ADD CONSTRAINT calendar_attendees_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.calendar_events(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.calendar_attendees
    ADD CONSTRAINT calendar_attendees_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.calendar_event_alarms
    ADD CONSTRAINT calendar_event_alarms_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.calendar_events(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_recurrence_id_fkey FOREIGN KEY (recurrence_id) REFERENCES public.calendar_recurrences(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.calendar_events
    ADD CONSTRAINT calendar_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.calendar_recurrences
    ADD CONSTRAINT calendar_recurrences_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.hr_attendance
    ADD CONSTRAINT hr_attendance_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.hr_attendance
    ADD CONSTRAINT hr_attendance_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_departments
    ADD CONSTRAINT fk_hr_departments_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_departments
    ADD CONSTRAINT fk_hr_departments_manager FOREIGN KEY (manager_id) REFERENCES public.hr_employees(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_departments
    ADD CONSTRAINT hr_departments_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.hr_departments(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT fk_hr_employees_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_expense_manager_id_fkey FOREIGN KEY (expense_manager_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.hr_jobs(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.hr_employees(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_employees
    ADD CONSTRAINT hr_employees_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expense_taxes
    ADD CONSTRAINT hr_expense_taxes_expense_id_fkey FOREIGN KEY (expense_id) REFERENCES public.hr_expenses(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_expense_taxes
    ADD CONSTRAINT hr_expense_taxes_tax_id_fkey FOREIGN KEY (tax_id) REFERENCES public.account_taxes(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_accounts(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_analytic_account_id_fkey FOREIGN KEY (analytic_account_id) REFERENCES public.account_analytic_account(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES public.res_currencies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_variants(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_split_origin_id_fkey FOREIGN KEY (split_origin_id) REFERENCES public.hr_expenses(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_expenses
    ADD CONSTRAINT hr_expenses_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_jobs
    ADD CONSTRAINT fk_hr_jobs_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_jobs
    ADD CONSTRAINT hr_jobs_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_leave_allocations
    ADD CONSTRAINT fk_hr_leave_allocations_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_leave_allocations
    ADD CONSTRAINT hr_leave_allocations_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_leave_requests
    ADD CONSTRAINT fk_hr_leave_requests_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.hr_leave_requests
    ADD CONSTRAINT hr_leave_requests_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.hr_employees(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_leave_requests
    ADD CONSTRAINT hr_leave_requests_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_overtime_lines
    ADD CONSTRAINT hr_overtime_lines_attendance_id_fkey FOREIGN KEY (attendance_id) REFERENCES public.hr_attendance(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.hr_overtime_lines
    ADD CONSTRAINT hr_overtime_lines_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.hr_overtime_lines
    ADD CONSTRAINT hr_overtime_lines_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.hr_overtime_rules
    ADD CONSTRAINT hr_overtime_rules_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.hr_work_entries
    ADD CONSTRAINT hr_work_entries_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.hr_work_entries
    ADD CONSTRAINT hr_work_entries_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.planning_roles
    ADD CONSTRAINT planning_roles_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.planning_shifts
    ADD CONSTRAINT planning_shifts_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.planning_shifts
    ADD CONSTRAINT planning_shifts_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.planning_shifts
    ADD CONSTRAINT planning_shifts_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.planning_roles(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.hr_jobs(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_recruiter_user_id_fkey FOREIGN KEY (recruiter_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_applicants
    ADD CONSTRAINT recruitment_applicants_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.recruitment_stages(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_interviews
    ADD CONSTRAINT recruitment_interviews_applicant_id_fkey FOREIGN KEY (applicant_id) REFERENCES public.recruitment_applicants(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.recruitment_interviews
    ADD CONSTRAINT recruitment_interviews_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.calendar_events(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_interviews
    ADD CONSTRAINT recruitment_interviews_interviewer_id_fkey FOREIGN KEY (interviewer_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.recruitment_stages
    ADD CONSTRAINT recruitment_stages_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.resource_calendars
    ADD CONSTRAINT resource_calendars_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.mrp_bom_lines
    ADD CONSTRAINT mrp_bom_lines_bom_id_fkey FOREIGN KEY (bom_id) REFERENCES public.mrp_boms(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_bom_lines
    ADD CONSTRAINT mrp_bom_lines_operation_id_fkey FOREIGN KEY (operation_id) REFERENCES public.mrp_routing_operations(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_bom_lines
    ADD CONSTRAINT mrp_bom_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_variants(id);

-- foreign key
ALTER TABLE ONLY public.mrp_bom_lines
    ADD CONSTRAINT mrp_bom_lines_uom_id_fkey FOREIGN KEY (uom_id) REFERENCES public.uom_uoms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_boms
    ADD CONSTRAINT mrp_boms_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_variants(id);

-- foreign key
ALTER TABLE ONLY public.mrp_boms
    ADD CONSTRAINT mrp_boms_uom_id_fkey FOREIGN KEY (uom_id) REFERENCES public.uom_uoms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_capacity_slots
    ADD CONSTRAINT mrp_capacity_slots_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_productions
    ADD CONSTRAINT mrp_productions_bom_id_fkey FOREIGN KEY (bom_id) REFERENCES public.mrp_boms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_productions
    ADD CONSTRAINT mrp_productions_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_variants(id);

-- foreign key
ALTER TABLE ONLY public.mrp_productions
    ADD CONSTRAINT mrp_productions_uom_id_fkey FOREIGN KEY (uom_id) REFERENCES public.uom_uoms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_quality_checks
    ADD CONSTRAINT mrp_quality_checks_point_id_fkey FOREIGN KEY (point_id) REFERENCES public.mrp_quality_points(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mrp_quality_checks
    ADD CONSTRAINT mrp_quality_checks_production_id_fkey FOREIGN KEY (production_id) REFERENCES public.mrp_productions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_quality_checks
    ADD CONSTRAINT mrp_quality_checks_workorder_id_fkey FOREIGN KEY (workorder_id) REFERENCES public.mrp_workorders(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_quality_points
    ADD CONSTRAINT mrp_quality_points_operation_id_fkey FOREIGN KEY (operation_id) REFERENCES public.mrp_routing_operations(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_quality_points
    ADD CONSTRAINT mrp_quality_points_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_routing_operations
    ADD CONSTRAINT mrp_routing_operations_bom_id_fkey FOREIGN KEY (bom_id) REFERENCES public.mrp_boms(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_routing_operations
    ADD CONSTRAINT mrp_routing_operations_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id);

-- foreign key
ALTER TABLE ONLY public.mrp_subcontracting_bom
    ADD CONSTRAINT mrp_subcontracting_bom_bom_id_fkey FOREIGN KEY (bom_id) REFERENCES public.mrp_boms(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_subcontracting_orders
    ADD CONSTRAINT mrp_subcontracting_orders_production_id_fkey FOREIGN KEY (production_id) REFERENCES public.mrp_productions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_bom_id_fkey FOREIGN KEY (bom_id) REFERENCES public.mrp_boms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_mo_id_fkey FOREIGN KEY (mo_id) REFERENCES public.mrp_productions(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_variants(id);

-- foreign key
ALTER TABLE ONLY public.mrp_unbuilds
    ADD CONSTRAINT mrp_unbuilds_uom_id_fkey FOREIGN KEY (uom_id) REFERENCES public.uom_uoms(id);

-- foreign key
ALTER TABLE ONLY public.mrp_workcenter_calendars
    ADD CONSTRAINT mrp_workcenter_calendars_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_workcenter_productivity
    ADD CONSTRAINT mrp_workcenter_productivity_loss_id_fkey FOREIGN KEY (loss_id) REFERENCES public.mrp_productivity_losses(id);

-- foreign key
ALTER TABLE ONLY public.mrp_workcenter_productivity
    ADD CONSTRAINT mrp_workcenter_productivity_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_workcenter_productivity
    ADD CONSTRAINT mrp_workcenter_productivity_workorder_id_fkey FOREIGN KEY (workorder_id) REFERENCES public.mrp_workorders(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mrp_workorder_time_logs
    ADD CONSTRAINT mrp_workorder_time_logs_workorder_id_fkey FOREIGN KEY (workorder_id) REFERENCES public.mrp_workorders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_workorders
    ADD CONSTRAINT mrp_workorders_operation_id_fkey FOREIGN KEY (operation_id) REFERENCES public.mrp_routing_operations(id);

-- foreign key
ALTER TABLE ONLY public.mrp_workorders
    ADD CONSTRAINT mrp_workorders_production_id_fkey FOREIGN KEY (production_id) REFERENCES public.mrp_productions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mrp_workorders
    ADD CONSTRAINT mrp_workorders_workcenter_id_fkey FOREIGN KEY (workcenter_id) REFERENCES public.mrp_workcenters(id);

-- foreign key
ALTER TABLE ONLY public.portal_users
    ADD CONSTRAINT portal_users_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.portal_users
    ADD CONSTRAINT portal_users_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_group_permissions
    ADD CONSTRAINT res_group_permissions_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.res_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_groups_implied_rel
    ADD CONSTRAINT res_groups_implied_rel_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.res_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_groups_implied_rel
    ADD CONSTRAINT res_groups_implied_rel_implied_group_id_fkey FOREIGN KEY (implied_group_id) REFERENCES public.res_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_groups_users_rel
    ADD CONSTRAINT res_groups_users_rel_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.res_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_groups_users_rel
    ADD CONSTRAINT res_groups_users_rel_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_record_rules
    ADD CONSTRAINT res_record_rules_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.res_groups(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.res_users
    ADD CONSTRAINT fk_res_users_company FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.res_users
    ADD CONSTRAINT fk_res_users_partner FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_analytic_account
    ADD CONSTRAINT account_analytic_account_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_analytic_account
    ADD CONSTRAINT account_analytic_account_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES public.account_analytic_plan(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_analytic_applicability
    ADD CONSTRAINT account_analytic_applicability_analytic_plan_id_fkey FOREIGN KEY (analytic_plan_id) REFERENCES public.account_analytic_plan(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.account_analytic_distribution_model
    ADD CONSTRAINT account_analytic_distribution_model_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_analytic_line
    ADD CONSTRAINT account_analytic_line_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.account_analytic_account(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.account_analytic_line
    ADD CONSTRAINT account_analytic_line_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.account_analytic_plan
    ADD CONSTRAINT account_analytic_plan_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.account_analytic_plan(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_assignation_logs
    ADD CONSTRAINT fleet_vehicle_assignation_logs_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_assignation_logs
    ADD CONSTRAINT fleet_vehicle_assignation_logs_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.fleet_vehicles(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_contracts
    ADD CONSTRAINT fleet_vehicle_log_contracts_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_contracts
    ADD CONSTRAINT fleet_vehicle_log_contracts_insurer_id_fkey FOREIGN KEY (insurer_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_contracts
    ADD CONSTRAINT fleet_vehicle_log_contracts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_contracts
    ADD CONSTRAINT fleet_vehicle_log_contracts_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.fleet_vehicles(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_services
    ADD CONSTRAINT fleet_vehicle_log_services_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_services
    ADD CONSTRAINT fleet_vehicle_log_services_service_type_id_fkey FOREIGN KEY (service_type_id) REFERENCES public.fleet_service_types(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_services
    ADD CONSTRAINT fleet_vehicle_log_services_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.fleet_vehicles(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_log_services
    ADD CONSTRAINT fleet_vehicle_log_services_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_models
    ADD CONSTRAINT fleet_vehicle_models_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES public.fleet_vehicle_brands(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_models
    ADD CONSTRAINT fleet_vehicle_models_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.fleet_vehicle_model_categories(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_odometers
    ADD CONSTRAINT fleet_vehicle_odometers_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.fleet_vehicles(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_tag_rel
    ADD CONSTRAINT fleet_vehicle_tag_rel_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.fleet_vehicle_tags(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicle_tag_rel
    ADD CONSTRAINT fleet_vehicle_tag_rel_vehicle_id_fkey FOREIGN KEY (vehicle_id) REFERENCES public.fleet_vehicles(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_future_driver_id_fkey FOREIGN KEY (future_driver_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_model_id_fkey FOREIGN KEY (model_id) REFERENCES public.fleet_vehicle_models(id);

-- foreign key
ALTER TABLE ONLY public.fleet_vehicles
    ADD CONSTRAINT fleet_vehicles_state_id_fkey FOREIGN KEY (state_id) REFERENCES public.fleet_vehicle_states(id);

-- foreign key
ALTER TABLE ONLY public.livechat_channel_users
    ADD CONSTRAINT livechat_channel_users_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES public.livechat_channels(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.livechat_channel_users
    ADD CONSTRAINT livechat_channel_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.livechat_channels
    ADD CONSTRAINT livechat_channels_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.livechat_messages
    ADD CONSTRAINT livechat_messages_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.livechat_sessions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.livechat_sessions
    ADD CONSTRAINT livechat_sessions_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES public.livechat_channels(id);

-- foreign key
ALTER TABLE ONLY public.livechat_sessions
    ADD CONSTRAINT livechat_sessions_operator_id_fkey FOREIGN KEY (operator_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.livechat_sessions
    ADD CONSTRAINT livechat_sessions_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.pos_cash_movements
    ADD CONSTRAINT pos_cash_movements_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.pos_sessions(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_cash_movements
    ADD CONSTRAINT pos_cash_movements_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.pos_config_payment_method_rel
    ADD CONSTRAINT pos_config_payment_method_rel_config_id_fkey FOREIGN KEY (config_id) REFERENCES public.pos_configs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_config_payment_method_rel
    ADD CONSTRAINT pos_config_payment_method_rel_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES public.pos_payment_methods(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_invoice_journal_id_fkey FOREIGN KEY (invoice_journal_id) REFERENCES public.account_journals(id);

-- foreign key
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id);

-- foreign key
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_stock_location_id_fkey FOREIGN KEY (stock_location_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.pos_configs
    ADD CONSTRAINT pos_configs_warehouse_id_fkey FOREIGN KEY (warehouse_id) REFERENCES public.stock_warehouses(id);

-- foreign key
ALTER TABLE ONLY public.pos_kitchen_ticket_lines
    ADD CONSTRAINT pos_kitchen_ticket_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.pos_kitchen_ticket_lines
    ADD CONSTRAINT pos_kitchen_ticket_lines_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES public.pos_kitchen_tickets(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_kitchen_tickets
    ADD CONSTRAINT pos_kitchen_tickets_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.pos_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_kitchen_tickets
    ADD CONSTRAINT pos_kitchen_tickets_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.restaurant_tables(id);

-- foreign key
ALTER TABLE ONLY public.pos_order_lines
    ADD CONSTRAINT pos_order_lines_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.pos_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_order_lines
    ADD CONSTRAINT pos_order_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.pos_sessions(id);

-- foreign key
ALTER TABLE ONLY public.pos_orders
    ADD CONSTRAINT pos_orders_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.pos_payment_methods
    ADD CONSTRAINT pos_payment_methods_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.pos_payment_methods
    ADD CONSTRAINT pos_payment_methods_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.account_journals(id);

-- foreign key
ALTER TABLE ONLY public.pos_payments
    ADD CONSTRAINT pos_payments_order_id_fkey FOREIGN KEY (order_id) REFERENCES public.pos_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.pos_payments
    ADD CONSTRAINT pos_payments_payment_method_id_fkey FOREIGN KEY (payment_method_id) REFERENCES public.pos_payment_methods(id);

-- foreign key
ALTER TABLE ONLY public.pos_payments
    ADD CONSTRAINT pos_payments_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.pos_sessions(id);

-- foreign key
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id);

-- foreign key
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_config_id_fkey FOREIGN KEY (config_id) REFERENCES public.pos_configs(id);

-- foreign key
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_stock_picking_id_fkey FOREIGN KEY (stock_picking_id) REFERENCES public.stock_pickings(id);

-- foreign key
ALTER TABLE ONLY public.pos_sessions
    ADD CONSTRAINT pos_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.pos_sync_batches
    ADD CONSTRAINT pos_sync_batches_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.pos_sessions(id);

-- foreign key
ALTER TABLE ONLY public.restaurant_floors
    ADD CONSTRAINT restaurant_floors_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.restaurant_floors
    ADD CONSTRAINT restaurant_floors_pos_config_id_fkey FOREIGN KEY (pos_config_id) REFERENCES public.pos_configs(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.restaurant_tables
    ADD CONSTRAINT restaurant_tables_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.restaurant_tables
    ADD CONSTRAINT restaurant_tables_floor_id_fkey FOREIGN KEY (floor_id) REFERENCES public.restaurant_floors(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_milestones
    ADD CONSTRAINT project_milestones_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_milestones
    ADD CONSTRAINT project_milestones_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project_projects(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_milestones
    ADD CONSTRAINT project_milestones_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_project_stages
    ADD CONSTRAINT project_project_stages_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_project_stages
    ADD CONSTRAINT project_project_stages_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_project_stages
    ADD CONSTRAINT project_project_stages_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_analytic_account_id_fkey FOREIGN KEY (analytic_account_id) REFERENCES public.account_analytic_account(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.project_project_stages(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.project_projects
    ADD CONSTRAINT project_projects_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_task_assignees
    ADD CONSTRAINT project_task_assignees_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.project_tasks(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_assignees
    ADD CONSTRAINT project_task_assignees_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_dependencies
    ADD CONSTRAINT project_task_dependencies_depends_on_task_id_fkey FOREIGN KEY (depends_on_task_id) REFERENCES public.project_tasks(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_dependencies
    ADD CONSTRAINT project_task_dependencies_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.project_tasks(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_tag_rel
    ADD CONSTRAINT project_task_tag_rel_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.project_task_tags(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_tag_rel
    ADD CONSTRAINT project_task_tag_rel_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.project_tasks(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_tags
    ADD CONSTRAINT project_task_tags_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_task_tags
    ADD CONSTRAINT project_task_tags_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_task_timers
    ADD CONSTRAINT project_task_timers_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.project_task_timers
    ADD CONSTRAINT project_task_timers_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.project_tasks(id);

-- foreign key
ALTER TABLE ONLY public.project_task_type_projects
    ADD CONSTRAINT project_task_type_projects_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project_projects(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_type_projects
    ADD CONSTRAINT project_task_type_projects_task_type_id_fkey FOREIGN KEY (task_type_id) REFERENCES public.project_task_types(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_types
    ADD CONSTRAINT project_task_types_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_task_types
    ADD CONSTRAINT project_task_types_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_task_types
    ADD CONSTRAINT project_task_types_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT fk_project_tasks_milestone FOREIGN KEY (milestone_id) REFERENCES public.project_milestones(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.project_tasks(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project_projects(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.project_task_types(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.project_tasks
    ADD CONSTRAINT project_tasks_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_analytic_account_id_fkey FOREIGN KEY (analytic_account_id) REFERENCES public.account_analytic_account(id);

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_project_id_fkey FOREIGN KEY (project_id) REFERENCES public.project_projects(id);

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.project_tasks(id);

-- foreign key
ALTER TABLE ONLY public.project_timesheets
    ADD CONSTRAINT project_timesheets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_sla_policies
    ADD CONSTRAINT helpdesk_sla_policies_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_sla_policies
    ADD CONSTRAINT helpdesk_sla_policies_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.helpdesk_teams(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.helpdesk_stages
    ADD CONSTRAINT helpdesk_stages_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_stages
    ADD CONSTRAINT helpdesk_stages_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.helpdesk_teams(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.helpdesk_teams
    ADD CONSTRAINT helpdesk_teams_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_assigned_user_id_fkey FOREIGN KEY (assigned_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_sale_order_id_fkey FOREIGN KEY (sale_order_id) REFERENCES public.sale_orders(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.helpdesk_stages(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_stock_picking_id_fkey FOREIGN KEY (stock_picking_id) REFERENCES public.stock_pickings(id);

-- foreign key
ALTER TABLE ONLY public.helpdesk_tickets
    ADD CONSTRAINT helpdesk_tickets_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.helpdesk_teams(id);

-- foreign key
ALTER TABLE ONLY public.knowledge_articles
    ADD CONSTRAINT knowledge_articles_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.knowledge_categories(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.knowledge_articles
    ADD CONSTRAINT knowledge_articles_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.knowledge_categories
    ADD CONSTRAINT knowledge_categories_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_activity_type_id_fkey FOREIGN KEY (activity_type_id) REFERENCES public.mail_activity_types(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_assigned_user_id_fkey FOREIGN KEY (assigned_user_id) REFERENCES public.res_users(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_activities
    ADD CONSTRAINT mail_activities_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_activity_types
    ADD CONSTRAINT mail_activity_types_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_activity_types
    ADD CONSTRAINT mail_activity_types_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_activity_types
    ADD CONSTRAINT mail_activity_types_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_email_queue
    ADD CONSTRAINT mail_email_queue_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_email_queue
    ADD CONSTRAINT mail_email_queue_notification_id_fkey FOREIGN KEY (notification_id) REFERENCES public.mail_notifications(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT mail_followers_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT mail_followers_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_followers
    ADD CONSTRAINT mail_followers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_followers_subtypes_rel
    ADD CONSTRAINT mail_followers_subtypes_rel_follower_id_fkey FOREIGN KEY (follower_id) REFERENCES public.mail_followers(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_followers_subtypes_rel
    ADD CONSTRAINT mail_followers_subtypes_rel_subtype_id_fkey FOREIGN KEY (subtype_id) REFERENCES public.mail_message_subtypes(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_activity_id_fkey FOREIGN KEY (activity_id) REFERENCES public.mail_activities(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.res_users(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.mail_messages(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_messages
    ADD CONSTRAINT mail_messages_subtype_id_fkey FOREIGN KEY (subtype_id) REFERENCES public.mail_message_subtypes(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.mail_notifications
    ADD CONSTRAINT mail_notifications_activity_id_fkey FOREIGN KEY (activity_id) REFERENCES public.mail_activities(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_notifications
    ADD CONSTRAINT mail_notifications_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id) ON DELETE RESTRICT;

-- foreign key
ALTER TABLE ONLY public.mail_notifications
    ADD CONSTRAINT mail_notifications_message_id_fkey FOREIGN KEY (message_id) REFERENCES public.mail_messages(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_notifications
    ADD CONSTRAINT mail_notifications_recipient_user_id_fkey FOREIGN KEY (recipient_user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mail_tracking_values
    ADD CONSTRAINT mail_tracking_values_message_id_fkey FOREIGN KEY (message_id) REFERENCES public.mail_messages(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mailing_contacts
    ADD CONSTRAINT mailing_contacts_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.mailing_contacts
    ADD CONSTRAINT mailing_contacts_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.mailing_list_contact_rel
    ADD CONSTRAINT mailing_list_contact_rel_contact_id_fkey FOREIGN KEY (contact_id) REFERENCES public.mailing_contacts(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mailing_list_contact_rel
    ADD CONSTRAINT mailing_list_contact_rel_list_id_fkey FOREIGN KEY (list_id) REFERENCES public.mailing_lists(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.mailing_lists
    ADD CONSTRAINT mailing_lists_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.mailing_traces
    ADD CONSTRAINT mailing_traces_contact_id_fkey FOREIGN KEY (contact_id) REFERENCES public.mailing_contacts(id);

-- foreign key
ALTER TABLE ONLY public.mailing_traces
    ADD CONSTRAINT mailing_traces_mailing_id_fkey FOREIGN KEY (mailing_id) REFERENCES public.mass_mailings(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.maintenance_equipment_categories(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.maintenance_teams(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment
    ADD CONSTRAINT maintenance_equipment_technician_user_id_fkey FOREIGN KEY (technician_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_equipment_categories
    ADD CONSTRAINT maintenance_equipment_categories_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.hr_departments(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES public.hr_employees(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_equipment_id_fkey FOREIGN KEY (equipment_id) REFERENCES public.maintenance_equipment(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.maintenance_stages(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.maintenance_teams(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_requests
    ADD CONSTRAINT maintenance_requests_technician_user_id_fkey FOREIGN KEY (technician_user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.maintenance_team_members
    ADD CONSTRAINT maintenance_team_members_team_id_fkey FOREIGN KEY (team_id) REFERENCES public.maintenance_teams(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.maintenance_team_members
    ADD CONSTRAINT maintenance_team_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.maintenance_teams
    ADD CONSTRAINT maintenance_teams_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.marketing_automation_activities
    ADD CONSTRAINT marketing_automation_activities_automation_id_fkey FOREIGN KEY (automation_id) REFERENCES public.marketing_automations(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.marketing_automation_activities
    ADD CONSTRAINT marketing_automation_activities_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.marketing_automation_activities(id) ON DELETE SET NULL;

-- foreign key
ALTER TABLE ONLY public.marketing_automations
    ADD CONSTRAINT marketing_automations_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.marketing_campaigns
    ADD CONSTRAINT marketing_campaigns_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.marketing_campaigns
    ADD CONSTRAINT marketing_campaigns_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.res_users(id);

-- foreign key
ALTER TABLE ONLY public.mass_mailings
    ADD CONSTRAINT mass_mailings_campaign_id_fkey FOREIGN KEY (campaign_id) REFERENCES public.marketing_campaigns(id);

-- foreign key
ALTER TABLE ONLY public.mass_mailings
    ADD CONSTRAINT mass_mailings_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.quality_alerts
    ADD CONSTRAINT quality_alerts_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.quality_alerts
    ADD CONSTRAINT quality_alerts_lot_id_fkey FOREIGN KEY (lot_id) REFERENCES public.stock_lots(id);

-- foreign key
ALTER TABLE ONLY public.quality_alerts
    ADD CONSTRAINT quality_alerts_picking_id_fkey FOREIGN KEY (picking_id) REFERENCES public.stock_pickings(id);

-- foreign key
ALTER TABLE ONLY public.quality_alerts
    ADD CONSTRAINT quality_alerts_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.quality_control_points
    ADD CONSTRAINT quality_control_points_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.product_categories(id);

-- foreign key
ALTER TABLE ONLY public.quality_control_points
    ADD CONSTRAINT quality_control_points_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.quality_control_points
    ADD CONSTRAINT quality_control_points_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.repair_order_lines
    ADD CONSTRAINT repair_order_lines_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.repair_order_lines
    ADD CONSTRAINT repair_order_lines_repair_id_fkey FOREIGN KEY (repair_id) REFERENCES public.repair_orders(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_account_move_id_fkey FOREIGN KEY (account_move_id) REFERENCES public.account_moves(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_location_dest_id_fkey FOREIGN KEY (location_dest_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_location_id_fkey FOREIGN KEY (location_id) REFERENCES public.stock_locations(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_partner_id_fkey FOREIGN KEY (partner_id) REFERENCES public.res_partners(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_product_id_fkey FOREIGN KEY (product_id) REFERENCES public.product_templates(id);

-- foreign key
ALTER TABLE ONLY public.repair_orders
    ADD CONSTRAINT repair_orders_product_lot_id_fkey FOREIGN KEY (product_lot_id) REFERENCES public.stock_lots(id);

-- foreign key
ALTER TABLE ONLY public.survey_questions
    ADD CONSTRAINT survey_questions_survey_id_fkey FOREIGN KEY (survey_id) REFERENCES public.survey_surveys(id) ON DELETE CASCADE;

-- foreign key
ALTER TABLE ONLY public.survey_surveys
    ADD CONSTRAINT survey_surveys_company_id_fkey FOREIGN KEY (company_id) REFERENCES public.res_companies(id);

-- index
CREATE INDEX idx_account_accounts_active ON public.account_accounts USING btree (active);

-- index
CREATE INDEX idx_account_accounts_code ON public.account_accounts USING btree (code);

-- index
CREATE INDEX idx_account_accounts_parent_id ON public.account_accounts USING btree (parent_id);

-- index
CREATE INDEX idx_account_accounts_type ON public.account_accounts USING btree (type);

-- index
CREATE INDEX idx_bsl_account_id ON public.account_bank_statement_lines USING btree (account_id);

-- index
CREATE INDEX idx_bsl_date ON public.account_bank_statement_lines USING btree (date);

-- index
CREATE INDEX idx_bsl_move_id ON public.account_bank_statement_lines USING btree (move_id);

-- index
CREATE INDEX idx_bsl_partner_id ON public.account_bank_statement_lines USING btree (partner_id);

-- index
CREATE INDEX idx_bsl_reconciled ON public.account_bank_statement_lines USING btree (reconciled);

-- index
CREATE INDEX idx_bsl_sequence ON public.account_bank_statement_lines USING btree (statement_id, sequence);

-- index
CREATE INDEX idx_bsl_statement_id ON public.account_bank_statement_lines USING btree (statement_id);

-- index
CREATE INDEX idx_account_bank_statements_active ON public.account_bank_statements USING btree (active);

-- index
CREATE INDEX idx_account_bank_statements_date ON public.account_bank_statements USING btree (date);

-- index
CREATE INDEX idx_account_bank_statements_journal_id ON public.account_bank_statements USING btree (journal_id);

-- index
CREATE INDEX idx_account_bank_statements_state ON public.account_bank_statements USING btree (state);

-- index
CREATE INDEX idx_account_cash_roundings_active ON public.account_cash_roundings USING btree (active);

-- index
CREATE INDEX idx_account_journals_active ON public.account_journals USING btree (active);

-- index
CREATE INDEX idx_account_journals_code ON public.account_journals USING btree (code);

-- index
CREATE INDEX idx_account_journals_type ON public.account_journals USING btree (type);

-- index
CREATE INDEX idx_account_move_lines_account_id ON public.account_move_lines USING btree (account_id);

-- index
CREATE INDEX idx_account_move_lines_matching_number ON public.account_move_lines USING btree (matching_number);

-- index
CREATE INDEX idx_account_move_lines_move_id ON public.account_move_lines USING btree (move_id);

-- index
CREATE INDEX idx_account_move_lines_partner_id ON public.account_move_lines USING btree (partner_id);

-- index
CREATE INDEX idx_account_move_lines_reconcile_state ON public.account_move_lines USING btree (reconcile, reconciled, partner_id, amount_residual);

-- index
CREATE INDEX idx_account_move_lines_statement_line_id ON public.account_move_lines USING btree (statement_line_id);

-- index
CREATE INDEX idx_account_moves_active ON public.account_moves USING btree (active);

-- index
CREATE INDEX idx_account_moves_date ON public.account_moves USING btree (date);

-- index
CREATE INDEX idx_account_moves_journal_id ON public.account_moves USING btree (journal_id);

-- index
CREATE INDEX idx_account_moves_move_type ON public.account_moves USING btree (move_type);

-- index
CREATE INDEX idx_account_moves_name ON public.account_moves USING btree (name);

-- index
CREATE INDEX idx_account_moves_partner_id ON public.account_moves USING btree (partner_id);

-- index
CREATE INDEX idx_account_moves_state ON public.account_moves USING btree (state);

-- index
CREATE INDEX idx_pr_credit_line ON public.account_partial_reconciles USING btree (credit_line_id);

-- index
CREATE INDEX idx_pr_credit_move ON public.account_partial_reconciles USING btree (credit_move_id);

-- index
CREATE INDEX idx_pr_debit_line ON public.account_partial_reconciles USING btree (debit_line_id);

-- index
CREATE INDEX idx_pr_debit_move ON public.account_partial_reconciles USING btree (debit_move_id);

-- index
CREATE INDEX idx_account_payment_reconciliations_invoice ON public.account_payment_reconciliations USING btree (invoice_id);

-- index
CREATE INDEX idx_account_payment_reconciliations_payment ON public.account_payment_reconciliations USING btree (payment_id);

-- index
CREATE INDEX idx_account_payment_term_lines_term_id ON public.account_payment_term_lines USING btree (payment_term_id);

-- index
CREATE INDEX idx_account_payment_terms_active ON public.account_payment_terms USING btree (active);

-- index
CREATE INDEX idx_account_payments_active ON public.account_payments USING btree (active);

-- index
CREATE INDEX idx_account_payments_date ON public.account_payments USING btree (date);

-- index
CREATE INDEX idx_account_payments_journal_id ON public.account_payments USING btree (journal_id);

-- index
CREATE INDEX idx_account_payments_name ON public.account_payments USING btree (name);

-- index
CREATE INDEX idx_account_payments_partner_id ON public.account_payments USING btree (partner_id);

-- index
CREATE INDEX idx_account_payments_state ON public.account_payments USING btree (state);

-- index
CREATE INDEX idx_account_payments_type ON public.account_payments USING btree (payment_type);

-- index
CREATE INDEX idx_rml_account_id ON public.account_reconcile_model_lines USING btree (account_id);

-- index
CREATE INDEX idx_rml_model_id ON public.account_reconcile_model_lines USING btree (reconcile_model_id);

-- index
CREATE INDEX idx_account_reconcile_models_active ON public.account_reconcile_models USING btree (active);

-- index
CREATE INDEX idx_account_reconcile_models_auto ON public.account_reconcile_models USING btree (is_auto_reconcile);

-- index
CREATE INDEX idx_account_taxes_active ON public.account_taxes USING btree (active);

-- index
CREATE INDEX idx_account_taxes_use ON public.account_taxes USING btree (type_tax_use);

-- index
CREATE INDEX idx_accounting_periods_dates ON public.accounting_periods USING btree (date_from, date_to);

-- index
CREATE INDEX idx_accounting_periods_state ON public.accounting_periods USING btree (state);

-- index
CREATE INDEX idx_payment_tokens_partner ON public.payment_tokens USING btree (partner_id, active);

-- index
CREATE INDEX idx_payment_transactions_idempotency ON public.payment_transactions USING btree (provider_id, idempotency_key) WHERE (idempotency_key IS NOT NULL);

-- index
CREATE INDEX idx_payment_transactions_invoice ON public.payment_transactions USING btree (invoice_id);

-- index
CREATE INDEX idx_payment_transactions_payment ON public.payment_transactions USING btree (payment_id);

-- index
CREATE INDEX idx_payment_transactions_provider_ref ON public.payment_transactions USING btree (provider_id, provider_reference);

-- index
CREATE INDEX idx_payment_transactions_sale_order ON public.payment_transactions USING btree (sale_order_id);

-- index
CREATE INDEX idx_res_currencies_active ON public.res_currencies USING btree (active);

-- index
CREATE INDEX idx_res_currencies_name ON public.res_currencies USING btree (name);

-- index
CREATE INDEX idx_currency_rates_company_id ON public.res_currency_rates USING btree (company_id);

-- index
CREATE INDEX idx_currency_rates_currency_id ON public.res_currency_rates USING btree (currency_id);

-- index
CREATE INDEX idx_currency_rates_date ON public.res_currency_rates USING btree (date);

-- index
CREATE UNIQUE INDEX uq_stock_account_config_cat ON public.stock_account_config USING btree (company_id, product_category_id) WHERE (product_category_id IS NOT NULL);

-- index
CREATE UNIQUE INDEX uq_stock_account_config_company_default ON public.stock_account_config USING btree (company_id) WHERE (product_category_id IS NULL);

-- index
CREATE INDEX idx_stock_landed_cost_lines_lc ON public.stock_landed_cost_lines USING btree (landed_cost_id);

-- index
CREATE INDEX idx_stock_landed_costs_pickings ON public.stock_landed_costs USING gin (picking_ids);

-- index
CREATE INDEX idx_stock_landed_costs_state ON public.stock_landed_costs USING btree (state);

-- index
CREATE INDEX idx_stock_locations_active ON public.stock_locations USING btree (active);

-- index
CREATE INDEX idx_stock_locations_parent_id ON public.stock_locations USING btree (parent_id);

-- index
CREATE INDEX idx_stock_locations_usage ON public.stock_locations USING btree (usage);

-- index
CREATE INDEX idx_stock_lots_product ON public.stock_lots USING btree (product_id);

-- index
CREATE INDEX idx_stock_move_lines_move ON public.stock_move_lines USING btree (move_id);

-- index
CREATE INDEX idx_stock_move_lines_product ON public.stock_move_lines USING btree (product_id);

-- index
CREATE INDEX idx_stock_move_production ON public.stock_moves USING btree (production_id);

-- index
CREATE INDEX idx_stock_move_production_fin ON public.stock_moves USING btree (production_finished_id);

-- index
CREATE INDEX idx_stock_moves_account_move_id ON public.stock_moves USING btree (account_move_id);

-- index
CREATE INDEX idx_stock_moves_location_dest_id ON public.stock_moves USING btree (location_dest_id);

-- index
CREATE INDEX idx_stock_moves_location_id ON public.stock_moves USING btree (location_id);

-- index
CREATE INDEX idx_stock_moves_picking_id ON public.stock_moves USING btree (picking_id);

-- index
CREATE INDEX idx_stock_moves_product_id ON public.stock_moves USING btree (product_id);

-- index
CREATE INDEX idx_stock_moves_state ON public.stock_moves USING btree (state);

-- index
CREATE INDEX idx_stock_moves_valuation ON public.stock_moves USING btree (value) WHERE (is_in OR is_out);

-- index
CREATE UNIQUE INDEX idx_stock_orderpoints_product_location_company ON public.stock_orderpoints USING btree (product_id, location_id, company_id) WHERE (active = true);

-- index
CREATE INDEX idx_stock_orderpoints_trigger ON public.stock_orderpoints USING btree (trigger) WHERE (active = true);

-- index
CREATE INDEX idx_stock_orderpoints_warehouse ON public.stock_orderpoints USING btree (warehouse_id);

-- index
CREATE INDEX idx_stock_pickings_active ON public.stock_pickings USING btree (active);

-- index
CREATE INDEX idx_stock_pickings_backorder_of_id ON public.stock_pickings USING btree (backorder_of_id) WHERE (backorder_of_id IS NOT NULL);

-- index
CREATE INDEX idx_stock_pickings_location_dest_id ON public.stock_pickings USING btree (location_dest_id);

-- index
CREATE INDEX idx_stock_pickings_location_id ON public.stock_pickings USING btree (location_id);

-- index
CREATE INDEX idx_stock_pickings_name ON public.stock_pickings USING btree (name);

-- index
CREATE INDEX idx_stock_pickings_origin ON public.stock_pickings USING btree (origin);

-- index
CREATE INDEX idx_stock_pickings_partner_id ON public.stock_pickings USING btree (partner_id);

-- index
CREATE INDEX idx_stock_pickings_picking_type ON public.stock_pickings USING btree (picking_type);

-- index
CREATE INDEX idx_stock_pickings_state ON public.stock_pickings USING btree (state);

-- index
CREATE INDEX idx_stock_quants_location_id ON public.stock_quants USING btree (location_id);

-- index
CREATE INDEX idx_stock_quants_product_id ON public.stock_quants USING btree (product_id);

-- index
CREATE INDEX idx_stock_rules_route_dest ON public.stock_rules USING btree (route_id, location_dest_id, active, sequence);

-- index
CREATE INDEX idx_stock_valuation_adj_landed_cost ON public.stock_valuation_adjustment_lines USING btree (landed_cost_id);

-- index
CREATE INDEX idx_stock_valuation_adj_move ON public.stock_valuation_adjustment_lines USING btree (move_id);

-- index
CREATE INDEX idx_stock_warehouses_active ON public.stock_warehouses USING btree (active);

-- index
CREATE INDEX idx_stock_warehouses_code ON public.stock_warehouses USING btree (code);

-- index
CREATE INDEX idx_attribute_values_attr_id ON public.product_attribute_values USING btree (attribute_id);

-- index
CREATE INDEX idx_product_categories_active ON public.product_categories USING btree (active);

-- index
CREATE INDEX idx_product_categories_name ON public.product_categories USING btree (name);

-- index
CREATE INDEX idx_product_categories_parent_id ON public.product_categories USING btree (parent_id);

-- index
CREATE INDEX idx_pricelist_items_category_id ON public.product_pricelist_items USING btree (category_id);

-- index
CREATE INDEX idx_pricelist_items_pricelist_id ON public.product_pricelist_items USING btree (pricelist_id);

-- index
CREATE INDEX idx_pricelist_items_template_id ON public.product_pricelist_items USING btree (template_id);

-- index
CREATE INDEX idx_pricelist_items_variant_id ON public.product_pricelist_items USING btree (variant_id);

-- index
CREATE INDEX idx_product_pricelists_active ON public.product_pricelists USING btree (active);

-- index
CREATE INDEX idx_product_templates_active ON public.product_templates USING btree (active);

-- index
CREATE INDEX idx_product_templates_barcode ON public.product_templates USING btree (barcode);

-- index
CREATE INDEX idx_product_templates_category_id ON public.product_templates USING btree (category_id);

-- index
CREATE INDEX idx_product_templates_internal_ref ON public.product_templates USING btree (internal_ref);

-- index
CREATE INDEX idx_product_templates_name ON public.product_templates USING btree (name);

-- index
CREATE INDEX idx_product_templates_sale_ok ON public.product_templates USING btree (sale_ok) WHERE (active = true);

-- index
CREATE INDEX idx_product_values_date ON public.product_values USING btree (date);

-- index
CREATE INDEX idx_product_values_move_id ON public.product_values USING btree (move_id);

-- index
CREATE INDEX idx_product_values_product_id ON public.product_values USING btree (product_id);

-- index
CREATE INDEX idx_product_variants_active ON public.product_variants USING btree (active);

-- index
CREATE INDEX idx_product_variants_barcode ON public.product_variants USING btree (barcode);

-- index
CREATE INDEX idx_product_variants_sku ON public.product_variants USING btree (sku);

-- index
CREATE INDEX idx_product_variants_template_id ON public.product_variants USING btree (template_id);

-- index
CREATE INDEX idx_uom_active ON public.uom_uoms USING btree (active);

-- index
CREATE INDEX idx_uom_category ON public.uom_uoms USING btree (category);

-- index
CREATE INDEX idx_uom_name ON public.uom_uoms USING btree (name);

-- index
CREATE INDEX idx_ecommerce_carts_session ON public.ecommerce_carts USING btree (session_uuid);

-- index
CREATE INDEX idx_ecommerce_carts_state_activity ON public.ecommerce_carts USING btree (state, last_activity_at);

-- index
CREATE INDEX idx_loyalty_card_history_card ON public.loyalty_card_history USING btree (card_id);

-- index
CREATE INDEX idx_loyalty_cards_active ON public.loyalty_cards USING btree (active);

-- index
CREATE INDEX idx_loyalty_cards_partner ON public.loyalty_cards USING btree (partner_id);

-- index
CREATE INDEX idx_loyalty_cards_program ON public.loyalty_cards USING btree (program_id);

-- index
CREATE INDEX idx_loyalty_mails_program ON public.loyalty_mails USING btree (program_id);

-- index
CREATE INDEX idx_loyalty_programs_active ON public.loyalty_programs USING btree (active);

-- index
CREATE INDEX idx_loyalty_programs_company ON public.loyalty_programs USING btree (company_id);

-- index
CREATE INDEX idx_loyalty_programs_type ON public.loyalty_programs USING btree (program_type);

-- index
CREATE INDEX idx_loyalty_rewards_program ON public.loyalty_rewards USING btree (program_id);

-- index
CREATE INDEX idx_loyalty_rules_program ON public.loyalty_rules USING btree (program_id);

-- index
CREATE UNIQUE INDEX uq_loyalty_rules_code ON public.loyalty_rules USING btree (code) WHERE (code IS NOT NULL);

-- index
CREATE INDEX idx_sale_order_coupon_points_coupon ON public.sale_order_coupon_points USING btree (coupon_id);

-- index
CREATE INDEX idx_sale_order_invoices_move_id ON public.sale_order_invoices USING btree (move_id);

-- index
CREATE INDEX idx_sale_order_lines_order_id ON public.sale_order_lines USING btree (order_id);

-- index
CREATE INDEX idx_sale_order_lines_product_id ON public.sale_order_lines USING btree (product_id);

-- index
CREATE INDEX idx_sale_orders_active ON public.sale_orders USING btree (active);

-- index
CREATE INDEX idx_sale_orders_date_order ON public.sale_orders USING btree (date_order);

-- index
CREATE INDEX idx_sale_orders_invoice_status ON public.sale_orders USING btree (invoice_status);

-- index
CREATE INDEX idx_sale_orders_name ON public.sale_orders USING btree (name);

-- index
CREATE INDEX idx_sale_orders_partner_id ON public.sale_orders USING btree (partner_id);

-- index
CREATE INDEX idx_sale_orders_state ON public.sale_orders USING btree (state);

-- index
CREATE INDEX idx_subscriptions_next_bill ON public.sale_subscriptions USING btree (state, next_billing_date);

-- index
CREATE INDEX idx_purchase_order_group_members_group ON public.purchase_order_group_members USING btree (group_id);

-- index
CREATE INDEX idx_purchase_order_invoices_move_id ON public.purchase_order_invoices USING btree (move_id);

-- index
CREATE INDEX idx_purchase_order_lines_order_id ON public.purchase_order_lines USING btree (order_id);

-- index
CREATE INDEX idx_purchase_order_lines_product_id ON public.purchase_order_lines USING btree (product_id);

-- index
CREATE INDEX idx_purchase_orders_active ON public.purchase_orders USING btree (active);

-- index
CREATE INDEX idx_purchase_orders_date_order ON public.purchase_orders USING btree (date_order);

-- index
CREATE INDEX idx_purchase_orders_invoice_status ON public.purchase_orders USING btree (invoice_status);

-- index
CREATE INDEX idx_purchase_orders_name ON public.purchase_orders USING btree (name);

-- index
CREATE INDEX idx_purchase_orders_partner_id ON public.purchase_orders USING btree (partner_id);

-- index
CREATE INDEX idx_purchase_orders_requisition_id ON public.purchase_orders USING btree (requisition_id);

-- index
CREATE INDEX idx_purchase_orders_state ON public.purchase_orders USING btree (state);

-- index
CREATE INDEX idx_purchase_supplier_infos_product_vendor ON public.purchase_supplier_infos USING btree (product_id, vendor_id);

-- index
CREATE INDEX idx_purchase_supplier_infos_requisition ON public.purchase_supplier_infos USING btree (requisition_id);

-- index
CREATE INDEX idx_purchase_requisition_lines_product_id ON public.purchase_requisition_lines USING btree (product_id);

-- index
CREATE INDEX idx_purchase_requisition_lines_requisition_id ON public.purchase_requisition_lines USING btree (requisition_id);

-- index
CREATE INDEX idx_purchase_requisition_lines_supplier ON public.purchase_requisition_lines USING btree (supplier_id);

-- index
CREATE INDEX idx_purchase_requisitions_active ON public.purchase_requisitions USING btree (active);

-- index
CREATE INDEX idx_purchase_requisitions_company_id ON public.purchase_requisitions USING btree (company_id);

-- index
CREATE INDEX idx_purchase_requisitions_date_start ON public.purchase_requisitions USING btree (date_start);

-- index
CREATE INDEX idx_purchase_requisitions_state ON public.purchase_requisitions USING btree (state);

-- index
CREATE INDEX idx_purchase_requisitions_type ON public.purchase_requisitions USING btree (requisition_type);

-- index
CREATE INDEX idx_purchase_requisitions_user_id ON public.purchase_requisitions USING btree (user_id);

-- index
CREATE INDEX idx_purchase_requisitions_vendor_id ON public.purchase_requisitions USING btree (vendor_id);

-- index
CREATE INDEX idx_crm_lead_tags_tag_id ON public.crm_lead_tags USING btree (tag_id);

-- index
CREATE INDEX idx_crm_leads_active ON public.crm_leads USING btree (active);

-- index
CREATE INDEX idx_crm_leads_date_deadline ON public.crm_leads USING btree (date_deadline);

-- index
CREATE INDEX idx_crm_leads_name ON public.crm_leads USING btree (name);

-- index
CREATE INDEX idx_crm_leads_partner_id ON public.crm_leads USING btree (partner_id);

-- index
CREATE INDEX idx_crm_leads_priority ON public.crm_leads USING btree (priority);

-- index
CREATE INDEX idx_crm_leads_salesperson_id ON public.crm_leads USING btree (salesperson_id);

-- index
CREATE INDEX idx_crm_leads_stage_id ON public.crm_leads USING btree (stage_id);

-- index
CREATE INDEX idx_crm_leads_type ON public.crm_leads USING btree (type);

-- index
CREATE INDEX idx_crm_lost_reasons_active ON public.crm_lost_reasons USING btree (active);

-- index
CREATE INDEX idx_crm_stages_active ON public.crm_stages USING btree (active);

-- index
CREATE INDEX idx_crm_stages_sequence ON public.crm_stages USING btree (sequence);

-- index
CREATE INDEX idx_ir_attachments_active ON public.ir_attachments USING btree (active);

-- index
CREATE INDEX idx_ir_attachments_checksum ON public.ir_attachments USING btree (checksum);

-- index
CREATE INDEX idx_ir_attachments_company_id ON public.ir_attachments USING btree (company_id);

-- index
CREATE INDEX idx_ir_attachments_expense_checksum ON public.ir_attachments USING btree (res_model, res_id, checksum) WHERE ((res_model)::text = 'hr.expense'::text);

-- index
CREATE INDEX idx_ir_attachments_name ON public.ir_attachments USING btree (name);

-- index
CREATE INDEX idx_ir_attachments_res_model_id ON public.ir_attachments USING btree (res_model, res_id);

-- index
CREATE INDEX idx_res_companies_active ON public.res_companies USING btree (active);

-- index
CREATE INDEX idx_res_companies_currency_id ON public.res_companies USING btree (currency_id);

-- index
CREATE INDEX idx_res_companies_name ON public.res_companies USING btree (name);

-- index
CREATE INDEX idx_res_companies_partner_id ON public.res_companies USING btree (partner_id);

-- index
CREATE INDEX idx_partners_active ON public.res_partners USING btree (active);

-- index
CREATE INDEX idx_partners_company_id ON public.res_partners USING btree (company_id);

-- index
CREATE INDEX idx_partners_email ON public.res_partners USING btree (email);

-- index
CREATE INDEX idx_partners_is_customer ON public.res_partners USING btree (is_customer) WHERE (active = true);

-- index
CREATE INDEX idx_partners_is_supplier ON public.res_partners USING btree (is_supplier) WHERE (active = true);

-- index
CREATE INDEX idx_partners_name ON public.res_partners USING btree (name);

-- index
CREATE INDEX idx_partners_parent_id ON public.res_partners USING btree (parent_id);

-- index
CREATE INDEX idx_appointment_bookings_staff_time ON public.appointment_bookings USING btree (staff_id, start_time, end_time);

-- index
CREATE INDEX idx_calendar_events_dates ON public.calendar_events USING btree (start_date, stop_date);

-- index
CREATE INDEX idx_calendar_events_ref ON public.calendar_events USING btree (res_model, res_id);

-- index
CREATE INDEX idx_calendar_events_user ON public.calendar_events USING btree (user_id);

-- index
CREATE INDEX idx_hr_attendance_check_in ON public.hr_attendance USING btree (check_in);

-- index
CREATE INDEX idx_hr_attendance_employee ON public.hr_attendance USING btree (employee_id);

-- index
CREATE INDEX idx_hr_departments_active ON public.hr_departments USING btree (active);

-- index
CREATE INDEX idx_hr_departments_manager_id ON public.hr_departments USING btree (manager_id);

-- index
CREATE INDEX idx_hr_departments_name ON public.hr_departments USING btree (name);

-- index
CREATE INDEX idx_hr_departments_parent_id ON public.hr_departments USING btree (parent_id);

-- index
CREATE INDEX idx_hr_employees_active ON public.hr_employees USING btree (active);

-- index
CREATE INDEX idx_hr_employees_dept ON public.hr_employees USING btree (department_id);

-- index
CREATE INDEX idx_hr_employees_email ON public.hr_employees USING btree (work_email);

-- index
CREATE INDEX idx_hr_employees_job ON public.hr_employees USING btree (job_id);

-- index
CREATE INDEX idx_hr_employees_manager ON public.hr_employees USING btree (manager_id);

-- index
CREATE INDEX idx_hr_employees_name ON public.hr_employees USING btree (name);

-- index
CREATE INDEX idx_hr_employees_partner_id ON public.hr_employees USING btree (partner_id);

-- index
CREATE INDEX idx_hr_expenses_company_id ON public.hr_expenses USING btree (company_id);

-- index
CREATE INDEX idx_hr_expenses_date ON public.hr_expenses USING btree (date);

-- index
CREATE INDEX idx_hr_expenses_duplicate ON public.hr_expenses USING btree (employee_id, date, total_amount) WHERE ((state)::text <> 'refused'::text);

-- index
CREATE INDEX idx_hr_expenses_employee_id ON public.hr_expenses USING btree (employee_id);

-- index
CREATE INDEX idx_hr_expenses_manager_id ON public.hr_expenses USING btree (manager_id);

-- index
CREATE INDEX idx_hr_expenses_manager_state ON public.hr_expenses USING btree (manager_id, state);

-- index
CREATE INDEX idx_hr_expenses_split_origin ON public.hr_expenses USING btree (split_origin_id);

-- index
CREATE INDEX idx_hr_expenses_state ON public.hr_expenses USING btree (state);

-- index
CREATE INDEX idx_hr_jobs_active ON public.hr_jobs USING btree (active);

-- index
CREATE INDEX idx_hr_jobs_dept ON public.hr_jobs USING btree (department_id);

-- index
CREATE INDEX idx_hr_jobs_name ON public.hr_jobs USING btree (name);

-- index
CREATE INDEX idx_hr_allocations_emp ON public.hr_leave_allocations USING btree (employee_id);

-- index
CREATE INDEX idx_hr_allocations_state ON public.hr_leave_allocations USING btree (state);

-- index
CREATE INDEX idx_hr_allocations_type_year ON public.hr_leave_allocations USING btree (leave_type, year);

-- index
CREATE INDEX idx_hr_leave_requests_dates ON public.hr_leave_requests USING btree (date_from, date_to);

-- index
CREATE INDEX idx_hr_leave_requests_emp ON public.hr_leave_requests USING btree (employee_id);

-- index
CREATE INDEX idx_hr_leave_requests_state ON public.hr_leave_requests USING btree (state);

-- index
CREATE INDEX idx_hr_leave_requests_type ON public.hr_leave_requests USING btree (leave_type);

-- index
CREATE INDEX idx_hr_overtime_lines_date ON public.hr_overtime_lines USING btree (date);

-- index
CREATE INDEX idx_hr_overtime_lines_employee ON public.hr_overtime_lines USING btree (employee_id);

-- index
CREATE INDEX idx_work_entries_emp_dates ON public.hr_work_entries USING btree (employee_id, date_start, date_stop);

-- index
CREATE INDEX idx_planning_shifts_time ON public.planning_shifts USING btree (employee_id, start_at, end_at);

-- index
CREATE INDEX idx_recruitment_applicants_job ON public.recruitment_applicants USING btree (job_id);

-- index
CREATE INDEX idx_recruitment_applicants_stage ON public.recruitment_applicants USING btree (stage_id);

-- index
CREATE INDEX idx_mrp_bom_lines_bom ON public.mrp_bom_lines USING btree (bom_id);

-- index
CREATE INDEX idx_mrp_boms_product ON public.mrp_boms USING btree (product_id);

-- index
CREATE INDEX idx_mrp_capacity_slots_wc_date ON public.mrp_capacity_slots USING btree (workcenter_id, date_start, date_end);

-- index
CREATE INDEX idx_mrp_prod_name ON public.mrp_productions USING btree (name);

-- index
CREATE INDEX idx_mrp_prod_product ON public.mrp_productions USING btree (product_id);

-- index
CREATE INDEX idx_mrp_prod_state ON public.mrp_productions USING btree (state);

-- index
CREATE INDEX idx_mrp_routing_ops_bom ON public.mrp_routing_operations USING btree (bom_id);

-- index
CREATE INDEX idx_mrp_unbuild_mo ON public.mrp_unbuilds USING btree (mo_id);

-- index
CREATE INDEX idx_mrp_unbuild_product ON public.mrp_unbuilds USING btree (product_id);

-- index
CREATE INDEX idx_mrp_wo_production ON public.mrp_workorders USING btree (production_id);

-- index
CREATE INDEX idx_mrp_wo_state ON public.mrp_workorders USING btree (state);

-- index
CREATE INDEX idx_res_group_permissions_group ON public.res_group_permissions USING btree (group_id);

-- index
CREATE INDEX idx_res_group_permissions_model ON public.res_group_permissions USING btree (model);

-- index
CREATE INDEX idx_res_groups_category ON public.res_groups USING btree (category);

-- index
CREATE INDEX idx_res_groups_name ON public.res_groups USING btree (name);

-- index
CREATE INDEX idx_res_groups_implied_rel_implied ON public.res_groups_implied_rel USING btree (implied_group_id);

-- index
CREATE INDEX idx_groups_users_rel_user ON public.res_groups_users_rel USING btree (user_id);

-- index
CREATE INDEX idx_res_record_rules_active ON public.res_record_rules USING btree (active);

-- index
CREATE INDEX idx_res_record_rules_group ON public.res_record_rules USING btree (group_id);

-- index
CREATE INDEX idx_res_record_rules_model ON public.res_record_rules USING btree (model);

-- index
CREATE INDEX idx_res_users_active ON public.res_users USING btree (active);

-- index
CREATE INDEX idx_res_users_company_id ON public.res_users USING btree (company_id);

-- index
CREATE INDEX idx_res_users_email ON public.res_users USING btree (email);

-- index
CREATE INDEX idx_res_users_login ON public.res_users USING btree (login);

-- index
CREATE INDEX idx_res_users_partner_id ON public.res_users USING btree (partner_id);

-- index
CREATE INDEX idx_analytic_account_active ON public.account_analytic_account USING btree (active);

-- index
CREATE UNIQUE INDEX idx_analytic_account_code ON public.account_analytic_account USING btree (code) WHERE (code IS NOT NULL);

-- index
CREATE INDEX idx_analytic_account_partner ON public.account_analytic_account USING btree (partner_id);

-- index
CREATE INDEX idx_analytic_account_plan ON public.account_analytic_account USING btree (plan_id);

-- index
CREATE INDEX idx_analytic_applicability_company ON public.account_analytic_applicability USING btree (company_id);

-- index
CREATE INDEX idx_analytic_applicability_domain ON public.account_analytic_applicability USING btree (business_domain);

-- index
CREATE INDEX idx_analytic_applicability_plan ON public.account_analytic_applicability USING btree (analytic_plan_id);

-- index
CREATE INDEX idx_analytic_dist_model_active ON public.account_analytic_distribution_model USING btree (active);

-- index
CREATE INDEX idx_analytic_dist_model_partner ON public.account_analytic_distribution_model USING btree (partner_id);

-- index
CREATE INDEX idx_analytic_dist_model_seq ON public.account_analytic_distribution_model USING btree (sequence);

-- index
CREATE INDEX idx_analytic_line_account ON public.account_analytic_line USING btree (account_id);

-- index
CREATE INDEX idx_analytic_line_company ON public.account_analytic_line USING btree (company_id);

-- index
CREATE INDEX idx_analytic_line_date ON public.account_analytic_line USING btree (date);

-- index
CREATE INDEX idx_analytic_line_dist_gin ON public.account_analytic_line USING gin (regexp_split_to_array((jsonb_path_query_array(analytic_distribution, '$.keyvalue()."key"'::jsonpath))::text, '\D+'::text)) WHERE (analytic_distribution IS NOT NULL);

-- index
CREATE INDEX idx_analytic_line_move ON public.account_analytic_line USING btree (move_line_id);

-- index
CREATE INDEX idx_analytic_line_source ON public.account_analytic_line USING btree (source);

-- index
CREATE INDEX idx_analytic_plan_active ON public.account_analytic_plan USING btree (active);

-- index
CREATE INDEX idx_analytic_plan_parent ON public.account_analytic_plan USING btree (parent_id);

-- index
CREATE INDEX idx_analytic_plan_root ON public.account_analytic_plan USING btree (root_id);

-- index
CREATE INDEX idx_fleet_services_state ON public.fleet_vehicle_log_services USING btree (state);

-- index
CREATE INDEX idx_fleet_services_type ON public.fleet_vehicle_log_services USING btree (service_type_id);

-- index
CREATE INDEX idx_fleet_vehicle_services_vehicle ON public.fleet_vehicle_log_services USING btree (vehicle_id);

-- index
CREATE INDEX idx_fleet_vehicle_odometers_vehicle ON public.fleet_vehicle_odometers USING btree (vehicle_id);

-- index
CREATE INDEX idx_fleet_vehicles_license ON public.fleet_vehicles USING btree (license_plate);

-- index
CREATE INDEX idx_fleet_vehicles_manager ON public.fleet_vehicles USING btree (manager_id);

-- index
CREATE INDEX idx_fleet_vehicles_state ON public.fleet_vehicles USING btree (state_id);

-- index
CREATE INDEX idx_livechat_sessions_visitor ON public.livechat_sessions USING btree (visitor_uuid);

-- index
CREATE INDEX idx_pos_orders_partner ON public.pos_orders USING btree (partner_id);

-- index
CREATE INDEX idx_pos_orders_session ON public.pos_orders USING btree (session_id);

-- index
CREATE INDEX idx_project_milestones_deadline ON public.project_milestones USING btree (date_deadline);

-- index
CREATE INDEX idx_project_milestones_project ON public.project_milestones USING btree (project_id);

-- index
CREATE INDEX idx_project_project_stages_company ON public.project_project_stages USING btree (company_id);

-- index
CREATE INDEX idx_project_project_stages_sequence ON public.project_project_stages USING btree (sequence, id);

-- index
CREATE INDEX idx_project_projects_active ON public.project_projects USING btree (active);

-- index
CREATE INDEX idx_project_projects_company ON public.project_projects USING btree (company_id);

-- index
CREATE INDEX idx_project_projects_manager ON public.project_projects USING btree (manager_id);

-- index
CREATE INDEX idx_project_projects_stage ON public.project_projects USING btree (stage_id);

-- index
CREATE INDEX idx_project_task_assignees_user ON public.project_task_assignees USING btree (user_id);

-- index
CREATE INDEX idx_project_task_dependencies_dependency ON public.project_task_dependencies USING btree (depends_on_task_id);

-- index
CREATE INDEX idx_project_task_tag_rel_tag ON public.project_task_tag_rel USING btree (tag_id);

-- index
CREATE UNIQUE INDEX idx_running_task_timer_employee ON public.project_task_timers USING btree (employee_id) WHERE (is_running = true);

-- index
CREATE INDEX idx_project_task_type_projects_project ON public.project_task_type_projects USING btree (project_id);

-- index
CREATE INDEX idx_project_task_types_company ON public.project_task_types USING btree (company_id);

-- index
CREATE INDEX idx_project_task_types_sequence ON public.project_task_types USING btree (sequence, id);

-- index
CREATE INDEX idx_project_tasks_company ON public.project_tasks USING btree (company_id);

-- index
CREATE INDEX idx_project_tasks_deadline ON public.project_tasks USING btree (date_deadline);

-- index
CREATE INDEX idx_project_tasks_parent ON public.project_tasks USING btree (parent_id);

-- index
CREATE INDEX idx_project_tasks_project ON public.project_tasks USING btree (project_id);

-- index
CREATE INDEX idx_project_tasks_stage ON public.project_tasks USING btree (stage_id);

-- index
CREATE INDEX idx_project_tasks_state ON public.project_tasks USING btree (state);

-- index
CREATE INDEX idx_timesheets_employee_date ON public.project_timesheets USING btree (employee_id, date);

-- index
CREATE INDEX idx_timesheets_project ON public.project_timesheets USING btree (project_id);

-- index
CREATE INDEX idx_timesheets_task ON public.project_timesheets USING btree (task_id);

-- index
CREATE INDEX idx_helpdesk_tickets_partner ON public.helpdesk_tickets USING btree (partner_id);

-- index
CREATE INDEX idx_helpdesk_tickets_team ON public.helpdesk_tickets USING btree (team_id);

-- index
CREATE INDEX idx_ir_config_parameters_company_id ON public.ir_config_parameters USING btree (company_id);

-- index
CREATE INDEX idx_ir_config_parameters_key ON public.ir_config_parameters USING btree (key);

-- index
CREATE INDEX idx_ir_sequences_active ON public.ir_sequences USING btree (active);

-- index
CREATE INDEX idx_ir_sequences_code ON public.ir_sequences USING btree (code);

-- index
CREATE INDEX idx_ir_sequences_company_id ON public.ir_sequences USING btree (company_id);

-- index
CREATE INDEX idx_ir_translation_lang ON public.ir_translation USING btree (lang);

-- index
CREATE INDEX idx_ir_translation_name_res_id ON public.ir_translation USING btree (name, res_id);

-- index
CREATE INDEX idx_mail_activities_deadline ON public.mail_activities USING btree (company_id, active, date_deadline);

-- index
CREATE INDEX idx_mail_activities_resource ON public.mail_activities USING btree (company_id, res_model, res_id, active, date_deadline, id);

-- index
CREATE INDEX idx_mail_activities_type_active ON public.mail_activities USING btree (activity_type_id, active);

-- index
CREATE INDEX idx_mail_activities_user_deadline ON public.mail_activities USING btree (company_id, assigned_user_id, active, date_deadline, id);

-- index
CREATE INDEX idx_mail_activity_types_active ON public.mail_activity_types USING btree (active);

-- index
CREATE INDEX idx_mail_activity_types_company ON public.mail_activity_types USING btree (company_id);

-- index
CREATE INDEX idx_mail_email_queue_company ON public.mail_email_queue USING btree (company_id, status, next_attempt_at);

-- index
CREATE INDEX idx_mail_email_queue_delivery ON public.mail_email_queue USING btree (status, next_attempt_at, id);

-- index
CREATE INDEX idx_mail_followers_resource ON public.mail_followers USING btree (res_model, res_id);

-- index
CREATE INDEX idx_mail_messages_activity ON public.mail_messages USING btree (activity_id);

-- index
CREATE INDEX idx_mail_messages_resource ON public.mail_messages USING btree (company_id, res_model, res_id, created_at, id);

-- index
CREATE INDEX idx_mail_notifications_activity ON public.mail_notifications USING btree (activity_id);

-- index
CREATE INDEX idx_mail_notifications_recipient ON public.mail_notifications USING btree (company_id, recipient_user_id, status, created_at, id);

-- index
CREATE INDEX idx_mail_tracking_message ON public.mail_tracking_values USING btree (message_id);

-- index
CREATE INDEX idx_mailing_traces_code ON public.mailing_traces USING btree (tracking_code);

-- index
CREATE INDEX idx_maintenance_equipment_company ON public.maintenance_equipment USING btree (company_id);

-- index
CREATE INDEX idx_maintenance_equipment_team ON public.maintenance_equipment USING btree (team_id);

-- index
CREATE INDEX idx_maintenance_requests_equipment ON public.maintenance_requests USING btree (equipment_id);

-- index
CREATE INDEX idx_maintenance_requests_kanban ON public.maintenance_requests USING btree (kanban_state, archived);

-- index
CREATE INDEX idx_maintenance_requests_recurring ON public.maintenance_requests USING btree (recurring_maintenance, archived);

-- index
CREATE INDEX idx_maintenance_requests_team ON public.maintenance_requests USING btree (team_id);

-- index
CREATE INDEX idx_transactions_created_at ON public.transactions USING btree (created_at DESC);

-- index
CREATE INDEX idx_transactions_type ON public.transactions USING btree (type);

-- trigger
CREATE TRIGGER trg_account_accounts_updated_at BEFORE UPDATE ON public.account_accounts FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_bank_statement_lines_updated_at BEFORE UPDATE ON public.account_bank_statement_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_bank_statements_updated_at BEFORE UPDATE ON public.account_bank_statements FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_cash_roundings_updated_at BEFORE UPDATE ON public.account_cash_roundings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_full_reconciles_updated_at BEFORE UPDATE ON public.account_full_reconciles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_journals_updated_at BEFORE UPDATE ON public.account_journals FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_move_lines_updated_at BEFORE UPDATE ON public.account_move_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_moves_updated_at BEFORE UPDATE ON public.account_moves FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_partial_reconciles_updated_at BEFORE UPDATE ON public.account_partial_reconciles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_payment_terms_updated_at BEFORE UPDATE ON public.account_payment_terms FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_payments_updated_at BEFORE UPDATE ON public.account_payments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_reconcile_models_updated_at BEFORE UPDATE ON public.account_reconcile_models FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_taxes_updated_at BEFORE UPDATE ON public.account_taxes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_accounting_periods_updated_at BEFORE UPDATE ON public.accounting_periods FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_currencies_updated_at BEFORE UPDATE ON public.res_currencies FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_currency_rates_updated_at BEFORE UPDATE ON public.res_currency_rates FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_landed_cost_lines_updated_at BEFORE UPDATE ON public.stock_landed_cost_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_landed_costs_updated_at BEFORE UPDATE ON public.stock_landed_costs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_locations_updated_at BEFORE UPDATE ON public.stock_locations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_moves_updated_at BEFORE UPDATE ON public.stock_moves FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_orderpoints_updated_at BEFORE UPDATE ON public.stock_orderpoints FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_pickings_updated_at BEFORE UPDATE ON public.stock_pickings FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_procurement_groups_updated_at BEFORE UPDATE ON public.stock_procurement_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_quants_updated_at BEFORE UPDATE ON public.stock_quants FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_valuation_adjustment_lines_updated_at BEFORE UPDATE ON public.stock_valuation_adjustment_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_stock_warehouses_updated_at BEFORE UPDATE ON public.stock_warehouses FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_categories_updated_at BEFORE UPDATE ON public.product_categories FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_pricelist_items_updated_at BEFORE UPDATE ON public.product_pricelist_items FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_pricelists_updated_at BEFORE UPDATE ON public.product_pricelists FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_templates_updated_at BEFORE UPDATE ON public.product_templates FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_values_updated_at BEFORE UPDATE ON public.product_values FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_product_variants_updated_at BEFORE UPDATE ON public.product_variants FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_uom_updated_at BEFORE UPDATE ON public.uom_uoms FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_loyalty_cards_updated_at BEFORE UPDATE ON public.loyalty_cards FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_loyalty_mails_updated_at BEFORE UPDATE ON public.loyalty_mails FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_loyalty_programs_updated_at BEFORE UPDATE ON public.loyalty_programs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_loyalty_rewards_updated_at BEFORE UPDATE ON public.loyalty_rewards FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_loyalty_rules_updated_at BEFORE UPDATE ON public.loyalty_rules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_sale_order_lines_updated_at BEFORE UPDATE ON public.sale_order_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_sale_orders_updated_at BEFORE UPDATE ON public.sale_orders FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_order_groups_updated_at BEFORE UPDATE ON public.purchase_order_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_order_lines_updated_at BEFORE UPDATE ON public.purchase_order_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_orders_updated_at BEFORE UPDATE ON public.purchase_orders FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_supplier_infos_updated_at BEFORE UPDATE ON public.purchase_supplier_infos FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_requisition_lines_updated_at BEFORE UPDATE ON public.purchase_requisition_lines FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_purchase_requisitions_updated_at BEFORE UPDATE ON public.purchase_requisitions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_crm_leads_updated_at BEFORE UPDATE ON public.crm_leads FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_crm_lost_reasons_updated_at BEFORE UPDATE ON public.crm_lost_reasons FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_crm_stages_updated_at BEFORE UPDATE ON public.crm_stages FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_crm_tags_updated_at BEFORE UPDATE ON public.crm_tags FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_ir_attachments_updated_at BEFORE UPDATE ON public.ir_attachments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_companies_updated_at BEFORE UPDATE ON public.res_companies FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_partners_updated_at BEFORE UPDATE ON public.res_partners FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_departments_updated_at BEFORE UPDATE ON public.hr_departments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_employees_updated_at BEFORE UPDATE ON public.hr_employees FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_expenses_updated_at BEFORE UPDATE ON public.hr_expenses FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_jobs_updated_at BEFORE UPDATE ON public.hr_jobs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_leave_allocations_updated_at BEFORE UPDATE ON public.hr_leave_allocations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_hr_leave_requests_updated_at BEFORE UPDATE ON public.hr_leave_requests FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_group_permissions_updated_at BEFORE UPDATE ON public.res_group_permissions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_groups_updated_at BEFORE UPDATE ON public.res_groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_record_rules_updated_at BEFORE UPDATE ON public.res_record_rules FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_res_users_updated_at BEFORE UPDATE ON public.res_users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_analytic_account_updated_at BEFORE UPDATE ON public.account_analytic_account FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_analytic_applicability_updated_at BEFORE UPDATE ON public.account_analytic_applicability FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_analytic_distribution_model_updated_at BEFORE UPDATE ON public.account_analytic_distribution_model FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_analytic_line_updated_at BEFORE UPDATE ON public.account_analytic_line FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_account_analytic_plan_updated_at BEFORE UPDATE ON public.account_analytic_plan FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_fleet_vehicle_log_contracts_updated_at BEFORE UPDATE ON public.fleet_vehicle_log_contracts FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_fleet_vehicle_log_services_updated_at BEFORE UPDATE ON public.fleet_vehicle_log_services FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_fleet_vehicles_updated_at BEFORE UPDATE ON public.fleet_vehicles FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_milestones_updated_at BEFORE UPDATE ON public.project_milestones FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_project_stages_updated_at BEFORE UPDATE ON public.project_project_stages FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_projects_updated_at BEFORE UPDATE ON public.project_projects FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_task_tags_updated_at BEFORE UPDATE ON public.project_task_tags FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_task_types_updated_at BEFORE UPDATE ON public.project_task_types FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_project_tasks_updated_at BEFORE UPDATE ON public.project_tasks FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_ir_config_parameters_updated_at BEFORE UPDATE ON public.ir_config_parameters FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_ir_sequences_updated_at BEFORE UPDATE ON public.ir_sequences FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_mail_activities_updated_at BEFORE UPDATE ON public.mail_activities FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_mail_activity_types_updated_at BEFORE UPDATE ON public.mail_activity_types FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_mail_email_queue_updated_at BEFORE UPDATE ON public.mail_email_queue FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_mail_notifications_updated_at BEFORE UPDATE ON public.mail_notifications FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_maintenance_equipment_updated_at BEFORE UPDATE ON public.maintenance_equipment FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- trigger
CREATE TRIGGER trg_maintenance_requests_updated_at BEFORE UPDATE ON public.maintenance_requests FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
